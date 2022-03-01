package bot

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.uber.org/zap"
)

func NewRecoverMiddleware(l *zap.Logger) *RecoverMiddleware {
	return &RecoverMiddleware{L: l}
}

type RecoverMiddleware struct {
	L *zap.Logger
}

func (m *RecoverMiddleware) Handle(b *tg.BotAPI, u *Update, next HandlerFunc) {
	defer func() {
		err := recover()
		if err != nil {
			m.L.Error("panic", zap.Any("recovered", err))
		}
	}()

	next(b, u)
}
