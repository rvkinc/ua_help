package bot

import (
	"context"
	"fmt"
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rvkinc/uasocial/internal/storage"
	"go.uber.org/zap"
)

type Config struct {
	Token string `yaml:"token"`
}

type Bot struct {
	Stack *Stack
	Api   *tg.BotAPI
	Ctx   context.Context
}

func New(ctx context.Context, config *Config, l *zap.Logger, d storage.Interface) (*Bot, error) {
	api, err := tg.NewBotAPI(config.Token)
	if err != nil {
		return nil, err
	}

	recovery := NewRecoverMiddleware(l)
	uinsert := NewUserInsertMiddleware(d, l)
	h := NewMessageHandler(api, l)

	stack := NewStack()
	stack.Use(recovery)
	stack.Use(uinsert)
	stack.UseHandler(h)

	return &Bot{
		Stack: stack,
		Api:   api,
		Ctx:   ctx,
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
		case <-b.Ctx.Done():
			return nil
		}
	}
}
