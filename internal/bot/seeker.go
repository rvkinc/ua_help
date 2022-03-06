package bot

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/google/uuid"
	"github.com/rvkinc/uasocial/internal/service"
)

const (
	maxSubscriptionsPerUser = 5
)

type seeker struct {
	category   *service.CategoryTranslated
	localities service.Localities
	locality   *service.Locality
}

func (m *MessageHandler) handleCmdMySubscriptions(u *Update) error {
	v := u.ctx.Value(userIDCtxKey)
	uid, ok := v.(uuid.UUID)
	if !ok {
		return fmt.Errorf("no user in context")
	}

	subs, err := m.Service.UserSubscriptions(u.ctx, uid)
	if err != nil {
		return fmt.Errorf("get user subscriptions: %w", err)
	}

	if len(subs) == 0 {
		_, err := m.Api.Send(tg.NewMessage(u.chatID(), m.Localize.Translate(errorNoSubscriptionsTr, UALang)))
		return err
	}

	for _, s := range subs {
		var b strings.Builder

		b.WriteString(fmt.Sprintf("%s %s\n", emojiTime, m.Localize.FormatDateTime(s.CreatedAt, UALang)))
		b.WriteString(fmt.Sprintf("%s %s\n", emojiLocation, s.Locality))
		b.WriteString(fmt.Sprintf("%s %s\n", emojiItem, s.Category))

		queryString := fmt.Sprintf("%s|%s", cmdMySubscriptions, s.ID.String())
		msg := tg.NewMessage(u.chatID(), b.String())
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = tg.InlineKeyboardMarkup{InlineKeyboard: [][]tg.InlineKeyboardButton{
			{
				{
					Text:         m.Localize.Translate(btnOptionDeleteTr, UALang),
					CallbackData: &queryString,
				},
			},
		}}

		_, err := m.Api.Send(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MessageHandler) handleSeekerUserRoleReply(u *Update) error {
	uid, err := u.userID()
	if err != nil {
		return err
	}

	count, err := m.Service.SubscriptionsCountByUser(u.ctx, uid)
	if err != nil {
		return err
	}

	if count >= maxSubscriptionsPerUser {
		m.dialogs.delete(u.chatID())
		msg := tg.NewMessage(u.chatID(), fmt.Sprintf(m.Localize.Translate(errorSubscriptionsLimitExceededTr, UALang), maxHelpsPerUser))
		msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
		_, err = m.Api.Send(msg)
		return err
	}

	d := m.dialogs.get(u.chatID())
	d.role = roleSeeker
	d.seeker = new(seeker)
	msg := tg.NewMessage(u.chatID(), m.Localize.Translate(seekerCategoryRequestTr, UALang))

	keyboardButtons := make([][]tg.KeyboardButton, 0)

	for _, category := range m.categories {
		if len(keyboardButtons) == 0 || len(keyboardButtons[len(keyboardButtons)-1]) == 2 {
			keyboardButtons = append(keyboardButtons, []tg.KeyboardButton{{Text: category.Name}})
			continue
		}
		keyboardButtons[len(keyboardButtons)-1] = append(keyboardButtons[len(keyboardButtons)-1], tg.KeyboardButton{Text: category.Name})
	}

	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		Keyboard:        keyboardButtons,
		OneTimeKeyboard: true,
		ResizeKeyboard:  true,
	}

	_, err = m.Api.Send(msg)
	if err != nil {
		return err
	}

	m.dialogs.get(u.chatID()).next = m.handleUserCategoryReply

	return nil
}

func (m *MessageHandler) handleUserCategoryReply(u *Update) error {
	d := m.dialogs.get(u.chatID())

	for i := range m.categories {
		if m.categories[i].Name == u.Message.Text {
			d.seeker.category = &m.categories[i]
		}
	}

	if d.seeker.category == nil {
		_, err := m.Api.Send(tg.NewMessage(u.chatID(), m.Localize.Translate(errorChooseOptionTr, UALang)))
		return err
	}

	msg := tg.NewMessage(u.chatID(), m.Localize.Translate(userLocalityRequestTr, UALang))
	msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
	_, err := m.Api.Send(msg)
	if err != nil {
		return err
	}

	m.dialogs.get(u.chatID()).next = m.handleSeekerLocalityTextReply

	return nil
}

func (m *MessageHandler) handleSeekerLocalityTextReply(u *Update) error {
	localities, err := m.Service.AutocompleteLocality(u.ctx, u.Message.Text)
	if err != nil {
		return err
	}

	if len(localities) == 0 {
		msg := tg.NewMessage(u.chatID(), m.Localize.Translate(errorPleaseTryAgainTr, UALang))
		msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
		_, err = m.Api.Send(msg)
		return err
	}

	keyboardButtons := make([][]tg.KeyboardButton, 0)

	for _, locality := range localities {
		fullLocality := fmt.Sprintf("%s, %s", locality.Name, locality.RegionName)
		keyboardButtons = append(keyboardButtons, []tg.KeyboardButton{{Text: fullLocality}})
	}

	m.dialogs.get(u.chatID()).seeker.localities = localities
	m.dialogs.get(u.chatID()).next = m.handleSeekerLocalityButtonReply

	msg := tg.NewMessage(u.chatID(), m.Localize.Translate(userLocalityReplyTr, UALang))
	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		OneTimeKeyboard: true,
		Keyboard:        keyboardButtons,
		ResizeKeyboard:  true,
	}

	_, err = m.Api.Send(msg)
	return err
}

func (m *MessageHandler) handleSeekerLocalityButtonReply(u *Update) error {
	d := m.dialogs.get(u.chatID())

	for _, l := range m.dialogs.get(u.chatID()).seeker.localities {
		if fmt.Sprintf("%s, %s", l.Name, l.RegionName) == u.Message.Text {
			d.seeker.locality = &l
			break
		}
	}

	if d.seeker.locality == nil {
		return m.handleSeekerLocalityTextReply(u)
	}

	_, err := m.Api.Send(tg.NewMessage(u.chatID(), m.Localize.Translate(seekerLookingForVolunteersTr, UALang)))
	if err != nil {
		m.L.Error("send message", zap.Error(err))
	}

	helps, err := m.Service.HelpsByCategoryLocation(u.ctx, d.seeker.locality.ID, d.seeker.category.ID)
	if err != nil {
		return err
	}

	if len(helps) == 0 {
		msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%s\n\n%s", m.Localize.Translate(seekerHelpsEmptyTr, UALang), m.Localize.Translate(seekerSubscriptionProposalTr, UALang)))
		msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
			Keyboard:        [][]tg.KeyboardButton{{{Text: m.Localize.Translate(btnOptionSubscribeTr, UALang)}}},
			OneTimeKeyboard: true,
			ResizeKeyboard:  true,
		}

		d.next = m.handleSeekerSubscriptionBtnReply
		_, err := m.Api.Send(msg)
		return err
	}

	for _, help := range helps {
		builder := strings.Builder{}
		builder.WriteString(fmt.Sprintf("%s %s\n", emojiLocation, help.Locality))
		builder.WriteString(fmt.Sprintf("%s %s\n", emojiTime, m.Localize.FormatDateTime(help.CreatedAt, UALang)))
		for _, c := range help.Categories {
			builder.WriteString(fmt.Sprintf("%s %s\n", emojiItem, c))
		}
		builder.WriteString(fmt.Sprintf("%s\n", help.Description))
		msg := tg.NewMessage(u.chatID(), builder.String())
		msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
		_, err = m.Api.Send(msg)
		if err != nil {
			return err
		}
	}

	msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%s\n", m.Localize.Translate(seekerSubscriptionProposalTr, UALang)))
	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		Keyboard:        [][]tg.KeyboardButton{{{Text: m.Localize.Translate(btnOptionSubscribeTr, UALang)}}},
		OneTimeKeyboard: true,
		ResizeKeyboard:  true,
	}

	d.next = m.handleSeekerSubscriptionBtnReply
	_, err = m.Api.Send(msg)
	return err
}

func (m *MessageHandler) handleSeekerSubscriptionBtnReply(u *Update) error {
	if u.Message.Text != m.Localize.Translate(btnOptionSubscribeTr, UALang) {
		return nil
	}

	uid, err := u.userID()
	if err != nil {
		return err
	}

	if err := m.Service.NewSubscription(u.ctx, service.CreateSubscription{
		CreatorID:  uid,
		CategoryID: m.dialogs.get(u.chatID()).seeker.category.ID,
		LocalityID: m.dialogs.get(u.chatID()).seeker.locality.ID,
	}); err != nil {
		if errors.Is(err, service.ErrAlreadyExists) {
			msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%s\n", m.Localize.Translate(seekerSubscriptionAlreadyExistsTr, UALang)))
			_, err := m.Api.Send(msg)
			return err
		}

		return err
	}

	m.dialogs.delete(u.chatID())
	msg := tg.NewMessage(u.chatID(), m.Localize.Translate(seekerSubscriptionCreateSuccessTr, UALang))
	msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
	_, err = m.Api.Send(msg)
	return err
}
