package bot

import (
	"context"
	"fmt"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

func NewMessageHandler() *MessageHandler {
	return &MessageHandler{}
}

type MessageHandler struct{}

func (m *MessageHandler) Handle(b *tg.BotAPI, u *Update) {
	fmt.Println("hello, i'm handling")
}

type Update struct {
	Context context.Context
	*tg.Update
}

func (u *Update) User() *tg.User {
	if u.CallbackQuery != nil {
		return u.CallbackQuery.From
	}
	return u.Message.From
}

func (u *Update) chatID() int64 {
	if u.CallbackQuery != nil {
		return u.CallbackQuery.Message.Chat.ID
	}
	return u.Message.Chat.ID
}

type Handler interface {
	Handle(*tg.BotAPI, *Update)
}

type HandlerFunc func(*tg.BotAPI, *Update)

func (f HandlerFunc) Handle(w *tg.BotAPI, r *Update) { f(w, r) }

type Middleware interface {
	Handle(writer *tg.BotAPI, request *Update, next HandlerFunc)
}

type mfunc func(rw *tg.BotAPI, r *Update, next HandlerFunc)

func (h mfunc) Handle(rw *tg.BotAPI, r *Update, next HandlerFunc) { h(rw, r, next) }

func NewStack() *Stack { return &Stack{} }

type Stack struct {
	middlewares []Middleware
	stack       *stack
}

func (s *Stack) Use(middleware Middleware) {
	s.middlewares = append(s.middlewares, middleware)
	s.stack = buildStack(s.middlewares)
}

func (s *Stack) UseHandler(handler Handler) {
	s.Use(wrap(handler))
}

func (s *Stack) Handle(rw *tg.BotAPI, r *Update) {
	s.stack.Handle(rw, r)
}

func wrap(h Handler) mfunc {
	return func(rw *tg.BotAPI, r *Update, _ HandlerFunc) { h.Handle(rw, r) }
}

type stack struct {
	middleware Middleware
	nextfn     func(*tg.BotAPI, *Update)
}

func (s *stack) Handle(rw *tg.BotAPI, r *Update) {
	s.middleware.Handle(rw, r, s.nextfn)
}

func buildStack(ms []Middleware) *stack {
	var next *stack

	switch {
	case len(ms) > 1:
		next = buildStack(ms[1:])
	case len(ms) == 0:
		fallthrough
	default:
		next = newNopStack()
	}

	return newStack(ms[0], next)
}

func newNopStack() *stack {
	return newStack(mfunc(func(rw *tg.BotAPI, r *Update, next HandlerFunc) {}), &stack{})
}

func newStack(mdl Middleware, next *stack) *stack {
	return &stack{middleware: mdl, nextfn: next.Handle}
}
