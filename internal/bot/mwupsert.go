package bot

import (
	"context"
	"fmt"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rvkinc/uasocial/internal/service"
	"go.uber.org/zap"
)

const userIDCtxKey = "user_id"

func NewUserUpsertMiddleware(ctx context.Context, l *zap.Logger, s *service.Service, api *tg.BotAPI, tr *Localizer) *UserUpsertMiddleware {
	return &UserUpsertMiddleware{
		L:        l,
		Localize: tr,
		Service:  s,
		Api:      api,

		ctx: ctx,
	}
}

type UserUpsertMiddleware struct {
	L        *zap.Logger
	Localize *Localizer
	Service  *service.Service
	Api      *tg.BotAPI

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
		msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%s\n", m.Localize.Translate(error500Tr, UALang)))
		_, _ = m.Api.Send(msg)
		return
	}

	u.ctx = context.WithValue(m.ctx, userIDCtxKey, user.ID)
	next(b, u)
}
