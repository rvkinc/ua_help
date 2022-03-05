package bot

import (
	"errors"
	"fmt"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/google/uuid"
	"github.com/rvkinc/uasocial/internal/service"
)

type seeker struct {
	category   service.CategoryTranslated
	localities service.Localities
	locality   service.Locality
}

func (m *MessageHandler) seekerUserRoleReply(chatID int64) error {
	d := m.state[chatID]
	d.role = roleSeeker
	d.seeker = new(seeker)
	msg := tg.NewMessage(chatID, m.Translator.Translate(userCategoryRequest, UALang))

	keyboardButtons := make([][]tg.KeyboardButton, 0)

	for _, category := range m.categories {
		if len(keyboardButtons) == 0 || len(keyboardButtons[len(keyboardButtons)-1]) == 2 {
			keyboardButtons = append(keyboardButtons, []tg.KeyboardButton{
				{
					Text: category.Name,
				},
			})
			continue
		}
		keyboardButtons[len(keyboardButtons)-1] = append(keyboardButtons[len(keyboardButtons)-1], tg.KeyboardButton{Text: category.Name})
	}

	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		OneTimeKeyboard: true,
		Keyboard:        keyboardButtons,
		ResizeKeyboard:  true,
	}

	_, err := m.Api.Send(msg)
	if err != nil {
		return err
	}

	m.state[chatID].next = m.handleUserCategoryReply

	return nil
}

func (m *MessageHandler) handleUserCategoryReply(u *Update) error {

	category := service.CategoryTranslated{
		ID:   m.categories.IDByName(u.Message.Text),
		Name: u.Message.Text,
	}

	m.state[u.chatID()].seeker.category = category

	msg := tg.NewMessage(u.chatID(), m.Translator.Translate(userLocalityRequestTranslation, UALang))

	_, err := m.Api.Send(msg)
	if err != nil {
		return err
	}

	m.state[u.chatID()].next = m.handleSeekerLocalityTextReply

	return nil
}

func (m *MessageHandler) handleSeekerLocalityTextReply(u *Update) error {
	msg := tg.NewMessage(u.chatID(), m.Translator.Translate(userLocalityReplyTranslation, UALang))

	localities, err := m.Service.AutocompleteLocality(u.ctx, u.Message.Text)
	if err != nil {
		return err
	}

	keyboardButtons := make([][]tg.KeyboardButton, 0)

	for _, locality := range localities {
		fullLocality := fmt.Sprintf("%s, %s", locality.Name, locality.RegionName)
		keyboardButtons = append(keyboardButtons, []tg.KeyboardButton{
			{
				Text: fullLocality,
			},
		})
	}

	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		OneTimeKeyboard: true,
		Keyboard:        keyboardButtons,
		ResizeKeyboard:  true,
	}

	_, err = m.Api.Send(msg)
	if err != nil {
		return err
	}

	m.state[u.chatID()].seeker.localities = localities
	m.state[u.chatID()].next = m.handleSeekerLocalityButtonReply

	return nil
}

func (m *MessageHandler) handleSeekerLocalityButtonReply(u *Update) error {
	vals := strings.Split(u.Message.Text, ", ")

	seeker := m.state[u.chatID()].seeker
	seeker.locality = seeker.localities.LocalityByNameRegion(vals[0], vals[1])

	helps, err := m.Service.HelpsByCategoryLocation(u.ctx, seeker.locality.ID, seeker.category.ID)
	if err != nil {
		return err
	}

	if len(helps) == 0 {
		_, err = m.Api.Send(tg.NewMessage(u.chatID(), m.Translator.Translate(helpsEmptyTranslation, UALang)))
		if err != nil {
			return err
		}

		return m.handleSeekerSubscriptionRequest(u)
	}

	for _, help := range helps {
		builder := strings.Builder{}
		builder.WriteString(fmt.Sprintf("%s: %s\n", m.Translator.Translate(helpCategoriesTranslation, UALang), strings.Join(help.Categories, ", ")))
		builder.WriteString(fmt.Sprintf("%s: %s\n", m.Translator.Translate(helpLocalityTranslation, UALang), help.Locality))
		builder.WriteString(fmt.Sprintf("%s: %s\n", m.Translator.Translate(helpCreateAtTranslation, UALang), help.CreatedAt))
		builder.WriteString(fmt.Sprintf("%s: \n%s", m.Translator.Translate(helpDetailsTranslation, UALang), help.Description))
		_, err = m.Api.Send(tg.NewMessage(u.chatID(), builder.String()))
		if err != nil {
			return err
		}
	}

	m.state[u.chatID()].next = m.handleSeekerSubscriptionRequest

	return nil
}

func (m *MessageHandler) handleSeekerSubscriptionRequest(u *Update) error {
	msg := tg.NewMessage(u.chatID(), m.Translator.Translate(subscriptionRequestTranslation, UALang))

	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		OneTimeKeyboard: true,
		Keyboard: [][]tg.KeyboardButton{
			{
				{
					Text: m.Translator.Translate(subscriptionButtonTranslation, UALang),
				},
			},
		},
		ResizeKeyboard: true,
	}

	_, err := m.Api.Send(msg)
	if err != nil {
		return err
	}

	m.state[u.chatID()].next = m.handleSeekerSubscriptionReply

	return nil
}

func (m *MessageHandler) handleSeekerSubscriptionReply(u *Update) error {
	if u.Message.Text != m.Translator.Translate(subscriptionButtonTranslation, UALang) {
		return nil
	}

	v := u.ctx.Value(UserIDCtxKey)
	uid, ok := v.(uuid.UUID)
	if !ok {
		return errors.New("can't get user id")
	}

	if err := m.Service.NewSubscription(u.ctx, service.CreateSubscription{
		CreatorID:  uid,
		CategoryID: m.state[u.chatID()].seeker.category.ID,
		LocalityID: m.state[u.chatID()].seeker.locality.ID,
	}); err != nil {
		return err
	}

	return nil
}
