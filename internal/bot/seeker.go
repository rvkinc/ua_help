package bot

import (
	"fmt"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (m *MessageHandler) handleSeekerCategory(u *Update) error {
	d := m.state[u.chatID()]

	c, ok := d.volunteer.invertCategoryButton(u.Message.Text)
	if !ok {
		_, err := m.Api.Send(tg.NewMessage(u.chatID(), m.Translator.Translate(errorChooseOption, UALang)))
		if err != nil {
			return err
		}
	}

	msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%d", c))
	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		OneTimeKeyboard: false,
		Keyboard:        d.volunteer.categoryKeyboardLayout(),
	}

	_, err := m.Api.Send(msg)
	return err
}
