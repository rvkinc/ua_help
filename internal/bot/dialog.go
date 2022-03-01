package bot

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.uber.org/zap"
)

type MessageHandler struct {
	Api *tg.BotAPI
	L   *zap.Logger
}

func NewMessageHandler(api *tg.BotAPI, l *zap.Logger) *MessageHandler {
	return &MessageHandler{
		Api: api,
		L:   l,
	}
}

func (m *MessageHandler) Handle(b *tg.BotAPI, u *Update) {

	_, err := m.Api.Send(tg.NewMessage(u.chatID(), "hello"))
	if err != nil {
		m.L.Error("send message", zap.Error(err))
	}

}
