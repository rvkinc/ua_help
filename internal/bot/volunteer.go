package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/google/uuid"
	"github.com/rvkinc/uasocial/internal/service"
	"go.uber.org/zap"
)

const (
	maxHelpsPerUser = 2
)

type volunteer struct {
	categories       []*category
	categoryKeyboard []*categoryCheckbox
	localities       service.Localities
	locality         service.Locality
	description      string
}

// command
func (m *MessageHandler) handleCmdMyHelp(u *Update) error {
	v := u.ctx.Value(userIDCtxKey)
	uid, ok := v.(uuid.UUID)
	if !ok {
		return fmt.Errorf("no user in context")
	}

	helps, err := m.Service.UserHelps(u.ctx, uid)
	if err != nil {
		return fmt.Errorf("get user helps: %w", err)
	}

	if len(helps) == 0 {
		msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%s\n\n%s", m.Localize.Translate(errorNoHelpsTr, UALang), m.Localize.Translate(navigationHintTr, UALang)))
		msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
		_, err = m.Api.Send(msg)
		return err
	}

	for _, h := range helps {
		var b strings.Builder
		b.WriteString(fmt.Sprintf("%s %s\n", emojiLocation, h.Locality))
		b.WriteString(fmt.Sprintf("%s %s\n", emojiTime, m.Localize.FormatDateTime(h.CreatedAt, UALang)))
		for _, c := range h.Categories {
			b.WriteString(fmt.Sprintf("%s %s\n", emojiItem, c))
		}
		b.WriteString(fmt.Sprintf("%s\n\n", h.Description))

		queryString := fmt.Sprintf("%s|%s", cmdMyHelp, h.ID.String())
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

func (m *MessageHandler) handleVolunteerUserRoleReply(u *Update) error {
	uid, err := u.userID()
	if err != nil {
		return err
	}

	count, err := m.Service.HelpsCountByUser(u.ctx, uid)
	if err != nil {
		return err
	}

	if count >= maxHelpsPerUser {
		m.dialogs.delete(u.chatID())
		msg := tg.NewMessage(u.chatID(), fmt.Sprintf(m.Localize.Translate(errorHelpsLimitExceededTr, UALang), maxHelpsPerUser))
		msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
		_, err = m.Api.Send(msg)
		return err
	}

	d := m.dialogs.get(u.chatID())
	d.role = roleVolunteer
	d.volunteer = new(volunteer)
	d.volunteer.categoryKeyboard = make([]*categoryCheckbox, 0, len(m.categories))
	for _, cc := range m.categories {
		d.volunteer.categoryKeyboard = append(d.volunteer.categoryKeyboard, &categoryCheckbox{
			category: category{uid: cc.ID, text: cc.Name},
			checked:  false,
		})
	}

	msg := tg.NewMessage(u.chatID(), m.Localize.Translate(volunteerSelectCategoriesRequestTr, UALang))
	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		OneTimeKeyboard: false,
		ResizeKeyboard:  true,
		Selective:       true,
		Keyboard:        d.volunteer.categoryKeyboardLayout(""),
	}

	_, err = m.Api.Send(msg)
	if err != nil {
		return err
	}

	m.dialogs.get(u.chatID()).next = m.handleVolunteerCategoryCheckboxReply
	return nil
}

func (m *MessageHandler) handleVolunteerCategoryCheckboxReply(u *Update) error {
	d := m.dialogs.get(u.chatID())
	nextBtnText := m.Localize.Translate(btnOptionNextTr, UALang)

	if u.Message.Text == nextBtnText && len(d.volunteer.categories) > 0 {
		msg := tg.NewMessage(u.chatID(), m.Localize.Translate(userLocalityRequestTr, UALang))
		msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
		_, err := m.Api.Send(msg)
		d.next = m.handleVolunteerLocalityTextReply
		return err
	}

	_, ok := d.volunteer.invertCategoryButton(u.Message.Text)
	if !ok {
		// garbage value
		_, err := m.Api.Send(tg.NewMessage(u.chatID(), m.Localize.Translate(errorChooseOptionTr, UALang)))
		if err != nil {
			return err
		}

		return nil
	}

	var txt string
	if len(d.volunteer.categories) != 0 {
		txt = fmt.Sprintf("%s:\n\n", m.Localize.Translate(volunteerChosenCategoriesHeaderTr, UALang))
		for _, c := range d.volunteer.categories {
			txt += fmt.Sprintf("%s %s\n", emojiItem, c.text)
		}
		txt += fmt.Sprintf("%s %s", m.Localize.Translate(volunteerChosenCategoriesFooterTr, UALang), m.Localize.Translate(btnOptionNextTr, UALang))
	} else {
		txt = m.Localize.Translate(errorChooseOptionTr, UALang)
	}

	// show or hide next button
	nextbtn := ""
	if len(d.volunteer.categories) > 0 {
		nextbtn = m.Localize.Translate(btnOptionNextTr, UALang)
	}

	msg := tg.NewMessage(u.chatID(), txt)
	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		OneTimeKeyboard: false,
		ResizeKeyboard:  true,
		Keyboard:        d.volunteer.categoryKeyboardLayout(nextbtn),
	}

	_, err := m.Api.Send(msg)
	return err
}

func (m *MessageHandler) handleVolunteerLocalityTextReply(u *Update) error {
	localities, err := m.Service.AutocompleteLocality(u.ctx, strings.Title(strings.ToLower(u.Message.Text)))
	if err != nil {
		return err
	}

	if len(localities) == 0 {
		msg := tg.NewMessage(u.chatID(), m.Localize.Translate(errorPleaseTryAgainTr, UALang))
		msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
		_, err := m.Api.Send(msg)
		return err
	}

	keyboardButtons := make([][]tg.KeyboardButton, 0)
	for _, locality := range localities {
		keyboardButtons = append(keyboardButtons, []tg.KeyboardButton{{Text: fmt.Sprintf("%s, %s", locality.Name, locality.RegionName)}})
	}

	msg := tg.NewMessage(u.chatID(), m.Localize.Translate(userLocalityReplyTr, UALang))
	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		Keyboard: keyboardButtons,
		// OneTimeKeyboard: true,
		ResizeKeyboard: true,
	}

	_, err = m.Api.Send(msg)
	if err != nil {
		return err
	}

	m.dialogs.get(u.chatID()).volunteer.localities = localities
	m.dialogs.get(u.chatID()).next = m.handleVolunteerLocalityButtonReply

	return nil
}

func (m *MessageHandler) handleVolunteerLocalityButtonReply(u *Update) error {
	d := m.dialogs.get(u.chatID())
	for _, l := range d.volunteer.localities {
		if fmt.Sprintf("%s, %s", l.Name, l.RegionName) == u.Message.Text {
			d.volunteer.locality = l
			d.next = m.handleVolunteerDescriptionTextReply
			msg := tg.NewMessage(u.chatID(), m.Localize.Translate(volunteerEnterDescriptionRequestTr, UALang))
			msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
			_, err := m.Api.Send(msg)
			return err
		}
	}

	return m.handleVolunteerLocalityTextReply(u)
}

func (m *MessageHandler) handleVolunteerDescriptionTextReply(u *Update) error {
	d := m.dialogs.get(u.chatID())
	d.volunteer.description = u.Message.Text

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s\n\n", m.Localize.Translate(volunteerSummaryHeaderTr, UALang)))
	b.WriteString(fmt.Sprintf("%s %s, %s\n", emojiLocation, d.volunteer.locality.Name, d.volunteer.locality.RegionName))
	b.WriteString(fmt.Sprintf("%s %s\n", emojiTime, m.Localize.FormatDateTime(time.Now(), UALang)))
	for _, c := range d.volunteer.categories {
		b.WriteString(fmt.Sprintf("%s %s\n", emojiItem, c.text))
	}
	b.WriteString(fmt.Sprintf("%s\n\n", d.volunteer.description))
	b.WriteString(fmt.Sprintf("%s", m.Localize.Translate(navigationHintTr, UALang)))

	uid, err := u.userID()
	if err != nil {
		return err
	}

	go func() {
		cids := make([]uuid.UUID, 0, len(d.volunteer.categories))
		for _, cs := range d.volunteer.categories {
			cids = append(cids, cs.uid)
		}

		err := m.Service.NewHelp(context.Background(), service.NewHelp{
			CreatorID:   uid,
			CategoryIDs: cids,
			LocalityID:  d.volunteer.locality.ID,
			Description: d.volunteer.description,
		})

		if err != nil {
			m.L.Error("create new help", zap.Error(err))
		}
	}()

	m.dialogs.delete(u.chatID())
	msg := tg.NewMessage(u.chatID(), b.String())
	msg.ParseMode = "HTML"
	_, err = m.Api.Send(msg)
	return err
}

func (v *volunteer) categoryKeyboardLayout(nextbtn string) [][]tg.KeyboardButton {
	layout := make([][]tg.KeyboardButton, 0, len(v.categoryKeyboard))
	for _, key := range v.categoryKeyboard {
		if len(layout) == 0 || len(layout[len(layout)-1]) == 2 {
			layout = append(layout, []tg.KeyboardButton{key.keyboardButton()})
			continue
		}

		layout[len(layout)-1] = append(layout[len(layout)-1], key.keyboardButton())
	}

	if nextbtn != "" {
		layout = append(layout, []tg.KeyboardButton{{Text: nextbtn}})
	}

	return layout
}

func (v *volunteer) invertCategoryButton(msg string) (uuid.UUID, bool) {
	for _, keyboard := range v.categoryKeyboard {
		if ok, checked := keyboard.invert(msg); ok {
			if checked {
				v.categories = append(v.categories, &category{
					uid:  keyboard.uid,
					text: keyboard.text,
				})
			} else {
				v.rmCategory(keyboard.uid)
			}
			return keyboard.uid, true
		}
	}

	return uuid.UUID{}, false
}

func (v *volunteer) rmCategory(uid uuid.UUID) {
	for i, x := range v.categories {
		if x.uid == uid {
			v.categories = append(v.categories[:i], v.categories[i+1:]...)
		}
	}
}

type (
	categoryCheckbox struct {
		category
		checked bool
	}

	category struct {
		uid  uuid.UUID
		text string
	}
)

func (b *categoryCheckbox) keyboardButton() tg.KeyboardButton {
	if b.checked {
		return tg.KeyboardButton{Text: emojiCheckbox + " " + b.text}
	}

	return tg.KeyboardButton{Text: b.text}
}

func (b *categoryCheckbox) invert(text string) (ok, checked bool) {
	if strings.Contains(text, b.text) {
		b.checked = !b.checked
		return true, b.checked
	}

	return false, false
}
