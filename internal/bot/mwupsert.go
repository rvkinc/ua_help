package bot

import (
	"context"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rvkinc/uasocial/internal/service"
	"go.uber.org/zap"
)

const (
	UserIDCtxKey = "user_id"
)

func NewUserUpsertMiddleware(ctx context.Context, l *zap.Logger, s *service.Service, api *tg.BotAPI, tr Translator) *UserUpsertMiddleware {
	return &UserUpsertMiddleware{
		L:       l,
		T:       tr,
		Service: s,
		Api:     api,

		ctx: ctx,
	}
}

type UserUpsertMiddleware struct {
	L       *zap.Logger
	T       Translator
	Service *service.Service
	Api     *tg.BotAPI

	ctx context.Context
}

func (m *UserUpsertMiddleware) Handle(b *tg.BotAPI, u *Update, next HandlerFunc) {
	user, err := m.Service.NewUser(m.ctx, &service.CreateUser{
		TgID:   u.userUUID().ID,
		ChatID: u.chatID(),
		Name:   u.userUUID().UserName,
	})

	if err != nil {
		m.L.Error("upsert user", zap.Error(err))
	}

	u.ctx = context.WithValue(m.ctx, UserIDCtxKey, user.ID)
	next(b, u)
}
