package bot

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (m *MessageHandler) seekerUserRoleReply(chatID int64) error {
	d := m.state[chatID]
	d.role = roleSeeker
	d.seeker = new(seeker)
	msg := tg.NewMessage(chatID, m.Translator.Translate(userRoleRequestTranslation, UALang))

	keyboardButtons := make([][]tg.KeyboardButton, len(m.categories))
	for i, category := range m.categories {
		keyboardButtons[i] = append(keyboardButtons[i], tg.KeyboardButton{Text: category.Name})
	}

	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		OneTimeKeyboard: true,
		Keyboard:        keyboardButtons,
	}

	_, err := m.Api.Send(msg)
	if err != nil {
		return err
	}

	m.state[chatID].next = m.handleSeekerCategory

	return nil
}

func (m *MessageHandler) handleSeekerCategory(u *Update) error {

	// msg := tg.NewMessage(u.chatID(), m.Translator.Translate(userRoleRequestTranslation, UALang))

	// m.Service.AutocompleteLocality(u.ctx)
	// categoryID := m.categories.UUIDByName(u.Message.Text)

	// m.Service.GetCategories()

	return nil
}
