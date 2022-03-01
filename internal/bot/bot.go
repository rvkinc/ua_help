package bot

import (
	"context"
	"fmt"
	"github.com/rvkinc/ua_help/internal/storage"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.uber.org/zap"
)

type Config struct {
	Token string `yaml:"token"`
}

// Run loads application dependencies, initializes bot instance and listening for updates
func Run(ctx context.Context, config *Config, log *zap.Logger, db storage.Interface) error {

	bot, err := tg.NewBotAPI(config.Token)
	if err != nil {
		return err
	}

	b := New(ctx, bot, log, db)

	u := tg.NewUpdate(0)
	u.Timeout = 60

	updch, err := bot.GetUpdatesChan(u)
	if err != nil {
		return fmt.Errorf("failed to get updates chan: %s", err)
	}

	for upd := range updch {
		func(u tg.Update) { b.Touch(&u) }(upd)
	}

	// sender := tsender.NewSender(&Provider{bot: bot})
	// go sender.Run(workers)
	// defer sender.Stop()
	//
	// return listen(config, upd, sender, recorder, searchesRepo, cookiesRepo)

	return nil
}

type B struct {
	Stack *Stack
	Bot   *tg.BotAPI
	Ctx   context.Context
}

func New(ctx context.Context, b *tg.BotAPI, l *zap.Logger, d storage.Interface) *B {
	recovery := NewRecoverMiddleware(l)
	// uinsert := NewUserInsertMiddleware(d, l) // todo: add

	stack := NewStack()
	stack.Use(recovery)
	// stack.Use(uinsert) // todo: add

	h := NewMessageHandler()
	stack.UseHandler(h)

	return &B{
		Stack: stack,
		Ctx:   ctx,
	}
}

func (b *B) Touch(upd *tg.Update) {
	b.Stack.Handle(b.Bot, &Update{
		Context: b.Ctx,
		Update:  upd,
	})
}
