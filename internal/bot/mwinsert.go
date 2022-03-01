package bot

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rvkinc/uasocial/internal/storage"
	"go.uber.org/zap"
)

func NewUserInsertMiddleware(d storage.Interface, l *zap.Logger) *UserInsertMiddleware {
	return &UserInsertMiddleware{
		L:  l,
		DB: d,
	}
}

type UserInsertMiddleware struct {
	L  *zap.Logger
	DB storage.Interface
}

func (m *UserInsertMiddleware) Handle(b *tg.BotAPI, u *Update, next HandlerFunc) {
	go func() {
		// insert unique user
	}()

	next(b, u)
}
