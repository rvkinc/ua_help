package bot

import (
	"context"
	"fmt"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rvkinc/uasocial/internal/service"
	"go.uber.org/zap"
)

type Config struct {
	Token string `yaml:"token"`
}

type Bot struct {
	Stack *Stack
	Api   *tg.BotAPI

	ctx context.Context
}

func New(ctx context.Context, config *Config, l *zap.Logger, s *service.Service) (*Bot, error) {
	api, err := tg.NewBotAPI(config.Token)
	if err != nil {
		return nil, err
	}

	tr, err := NewTranslator()
	if err != nil {
		return nil, err
	}

	recoveryMiddleware := NewRecoverMiddleware(l)
	upsertMiddleware := NewUserUpsertMiddleware(ctx, l, s, api, tr)
	h, err := NewMessageHandler(ctx, api, l, s, tr)
	if err != nil {
		return nil, err
	}

	stack := NewStack()
	stack.Use(recoveryMiddleware)
	stack.Use(upsertMiddleware)
	stack.UseHandler(h)

	return &Bot{
		Stack: stack,
		Api:   api,
		ctx:   ctx,
	}, nil
}

func (b *Bot) Run() error {
	u := tg.NewUpdate(0)
	u.Timeout = 60

	updch, err := b.Api.GetUpdatesChan(u)
	if err != nil {
		return fmt.Errorf("failed to get updates chan: %s", err)
	}

	for {
		select {
		case upd := <-updch:
			go func(u tg.Update) { b.Stack.Handle(b.Api, &Update{Update: &u}) }(upd)
		case <-b.ctx.Done():
			return nil
		}
	}
}
