package bot

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"strings"

	"github.com/rvkinc/uasocial/internal/service"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.uber.org/zap"
)

const (
	roleVolunteer role = iota + 1
	roleSeeker
)

const (
	cmdStart = "start"
)

const (
	emojiCheckbox = "✅"
	emojiItem     = "▪️"
)

type MessageHandler struct {
	Api        *tg.BotAPI
	L          *zap.Logger
	Translator Translator
	Service    *service.Service

	state      map[int64]*dialog
	categories service.CategoriesTranslated
}

func NewMessageHandler(ctx context.Context, api *tg.BotAPI, l *zap.Logger, s *service.Service, tr Translator) (*MessageHandler, error) {
	m := &MessageHandler{
		Api:        api,
		L:          l,
		Translator: tr,
		Service:    s,

		state: make(map[int64]*dialog),
	}

	categories, err := s.GetCategories(ctx)
	if err != nil {
		return nil, err
	}

	m.categories = categories.Translate(UALang)

	return m, nil
}

type (
	role int

	handler func(*Update) error

	volunteer struct {
		categories       []*category
		categoryKeyboard []*categoryCheckbox
	}
)

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

type seeker struct {
}

type dialog struct {
	role role
	next handler

	// either one is populated during the dialog
	volunteer *volunteer
	seeker    *seeker
}

func (m *MessageHandler) Handle(_ *tg.BotAPI, u *Update) {
	if u.Message != nil && u.Message.IsCommand() {
		switch u.Message.Command() {
		case cmdStart:
			err := m.userRoleRequest(u)
			if err != nil {
				m.L.Error("handle start cmd", zap.Error(err))
			}
			return
		default:
			_, _ = m.Api.Send(tg.NewMessage(u.chatID(), "Error"))
			return
		}
	}

	dialog, ok := m.state[u.chatID()]
	if !ok {
		err := m.userRoleRequest(u)
		if err != nil {
			m.L.Error("handle user role request", zap.Error(err))
		}
		return
	}

	err := dialog.next(u)
	if err != nil {
		m.L.Error("handle request", zap.Error(err))
	}
}

func (m *MessageHandler) userRoleRequest(u *Update) error {
	msg := tg.NewMessage(u.chatID(), m.Translator.Translate(userRoleRequestTranslation, UALang))
	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		OneTimeKeyboard: false,
		ResizeKeyboard:  true,
		Keyboard: [][]tg.KeyboardButton{
			{tg.KeyboardButton{Text: m.Translator.Translate(btnOptionUserRoleSeeker, UALang)}},
			{tg.KeyboardButton{Text: m.Translator.Translate(btnOptionUserRoleVolunteer, UALang)}},
		},
	}

	_, err := m.Api.Send(msg)
	if err != nil {
		return err
	}

	m.state[u.chatID()] = &dialog{next: m.handleUserRoleReply}
	return nil
}

func (m *MessageHandler) handleUserRoleReply(u *Update) error {
	switch u.Message.Text {
	case m.Translator.Translate(btnOptionUserRoleSeeker, UALang):
		return m.seekerUserRoleReply(u.chatID())
	case m.Translator.Translate(btnOptionUserRoleVolunteer, UALang):
		d := m.state[u.chatID()]
		d.role = roleVolunteer
		d.volunteer = new(volunteer)
		d.volunteer.categoryKeyboard = make([]*categoryCheckbox, 0, len(m.categories))
		for _, cc := range m.categories {
			d.volunteer.categoryKeyboard = append(d.volunteer.categoryKeyboard, &categoryCheckbox{
				category: category{uid: cc.ID, text: cc.Name},
				checked:  false,
			})
		}

		msg := tg.NewMessage(u.chatID(), m.Translator.Translate(userRoleRequestTranslation, UALang))
		msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
			OneTimeKeyboard: false,
			ResizeKeyboard:  true,
			Selective:       true,
			Keyboard:        d.volunteer.categoryKeyboardLayout(""),
		}

		_, err := m.Api.Send(msg)
		if err != nil {
			return err
		}

		m.state[u.chatID()].next = m.handleVolunteerCategoryCheckbox
	default:
		_, err := m.Api.Send(tg.NewMessage(u.chatID(), m.Translator.Translate(errorChooseOption, UALang)))
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MessageHandler) handleVolunteerCategoryCheckbox(u *Update) error {
	d := m.state[u.chatID()]
	nextBtnText := m.Translator.Translate(nextButtonTr, UALang)

	if u.Message.Text == nextBtnText && len(d.volunteer.categories) > 0 {
		// todo: set next handler
		return nil
	}

	_, ok := d.volunteer.invertCategoryButton(u.Message.Text)
	if !ok {
		// garbage value
		_, err := m.Api.Send(tg.NewMessage(u.chatID(), m.Translator.Translate(errorChooseOption, UALang)))
		if err != nil {
			return err
		}

		return nil
	}

	var txt string
	if len(d.volunteer.categories) != 0 {
		txt = fmt.Sprintf("%s:\n\n", m.Translator.Translate(volunteerChosenCategoriesHeaderTr, UALang))
		for _, c := range d.volunteer.categories {
			txt += fmt.Sprintf("%s %s\n", emojiItem, c.text)
		}
		txt += fmt.Sprintf("%s %s", m.Translator.Translate(volunteerChosenCategoriesFooterTr, UALang), m.Translator.Translate(nextButtonTr, UALang))
	} else {
		txt = m.Translator.Translate(errorChooseOption, UALang)
	}

	// show or hide next button
	nextbtn := ""
	if len(d.volunteer.categories) > 0 {
		nextbtn = m.Translator.Translate(nextButtonTr, UALang)
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
