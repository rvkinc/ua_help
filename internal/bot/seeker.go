package bot

import (
	"fmt"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
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
	seeker.localities = nil

	return nil
}
