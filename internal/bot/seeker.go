package bot

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rvkinc/uasocial/internal/service"
)

type seeker struct {
	category service.CategoryTranslated
	locality service.Locality
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

	m.state[u.chatID()].next = m.handleUserLocalityReply

	return nil
}

func (m *MessageHandler) handleSeekerCategory(u *Update) error {

	// msg := tg.NewMessage(u.chatID(), m.Translator.Translate(userRoleRequestTranslation, UALang))

	// m.Service.AutocompleteLocality(u.ctx)
	// categoryID := m.categories.UUIDByName(u.Message.Text)

	// m.Service.GetCategories()

	return nil
}
