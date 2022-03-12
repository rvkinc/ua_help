package bot

import (
	"context"
	"fmt"
	"strings"
	"sync"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/google/uuid"
	"github.com/rvkinc/uasocial/internal/service"
	"go.uber.org/zap"
)

const (
	roleVolunteer role = iota + 1
	roleSeeker
)

const (
	cmdStart           = "start"
	cmdMyHelp          = "my_help"
	cmdMySubscriptions = "my_subscriptions"
	cmdSupport         = "support"

	cqHelpsBySubscription = "hepls_by_subscription"
)

const (
	emojiCheckbox = "✅"
	emojiItem     = "🔸"
	emojiLocation = "🏡"
	emojiTime     = "⏱"
)

const adminTgID = 386274487

type (
	role    int
	handler func(*Update) error
	dialog  struct {
		role role
		next handler

		// either one is populated during the dialog
		volunteer *volunteer
		seeker    *seeker
	}
)

type dialogs struct {
	mu    *sync.Mutex
	state map[int64]*dialog
}

func (d *dialogs) set(dialog *dialog, chatID int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.state[chatID] = dialog
}

func (d *dialogs) get(chatID int64) *dialog {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.state[chatID]
}

func (d *dialogs) delete(chatID int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.state, chatID)
}

type MessageHandler struct {
	Api      *tg.BotAPI
	L        *zap.Logger
	Localize *Localizer
	Service  *service.Service

	dialogs    *dialogs
	categories service.CategoriesTranslated
}

func NewMessageHandler(ctx context.Context, api *tg.BotAPI, l *zap.Logger, s *service.Service, tr *Localizer) (*MessageHandler, error) {
	m := &MessageHandler{
		Api:      api,
		L:        l,
		Localize: tr,
		Service:  s,
		dialogs:  &dialogs{mu: &sync.Mutex{}, state: make(map[int64]*dialog)},
	}

	categories, err := s.GetCategories(ctx)
	if err != nil {
		return nil, err
	}

	m.categories = categories.Translate(UALang)
	go m.listenSubscriptionUpdates(ctx)
	return m, nil
}

func (m *MessageHandler) listenSubscriptionUpdates(ctx context.Context) {
	for {
		select {
		case upd := <-m.Service.Subscriptions():
			for _, u := range upd {
				var b strings.Builder
				b.WriteString(fmt.Sprintf("%s\n\n", m.Localize.Translate(seekerSubscriptionUpdateHeaderTr, UALang)))
				b.WriteString(fmt.Sprintf("%s %s\n", emojiLocation, u.Locality))
				b.WriteString(fmt.Sprintf("%s %s\n", emojiTime, m.Localize.FormatDateTime(u.CreatedAt, UALang)))
				for _, c := range u.Categories {
					b.WriteString(fmt.Sprintf("%s %s\n", emojiItem, c))
				}
				b.WriteString(fmt.Sprintf("%s\n\n", u.Description))
				msg := tg.NewMessage(u.ChatID, b.String())
				_, err := m.Api.Send(msg)
				if err != nil {
					m.L.Error("send subscription update", zap.Error(err))
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *MessageHandler) Handle(_ *tg.BotAPI, u *Update) {
	if u.CallbackQuery != nil {
		err := m.handleCallbackQuery(u)
		if err != nil {
			m.L.Error("handle callback query", zap.Error(err))
		}
		return
	}

	if u.Message != nil && u.Message.IsCommand() {
		m.dialogs.delete(u.chatID())
		switch u.Message.Command() {
		case cmdStart:
			err := m.handleCmdStart(u)
			if err != nil {
				m.L.Error("handle start cmd", zap.Error(err))
			}
			return
		case cmdMyHelp:
			err := m.handleCmdMyHelp(u)
			if err != nil {
				m.L.Error("handle cmd", zap.Error(err), zap.String("cmd", cmdMyHelp))
			}
			return
		case cmdMySubscriptions:
			err := m.handleCmdMySubscriptions(u)
			if err != nil {
				m.L.Error("handle cmd", zap.Error(err), zap.String("cmd", cmdMyHelp))
			}
			return
		case cmdSupport:
			err := m.handleCmdSupport(u)
			if err != nil {
				m.L.Error("handle cmd", zap.Error(err), zap.String("cmd", cmdMyHelp))
			}
			return
		}
	}

	if u.Message.Text == m.Localize.Translate(btnOptionCancelTr, UALang) {
		m.dialogs.delete(u.chatID())
		msg := tg.NewMessage(u.chatID(), m.Localize.Translate(navigationHintTr, UALang))
		msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
		_, err := m.Api.Send(msg)
		if err != nil {
			m.L.Error("handle cancel:", zap.Error(err))
		}
		return
	}

	dialog := m.dialogs.get(u.chatID())
	if dialog == nil {
		err := m.handleCmdStart(u)
		if err != nil {
			m.L.Error("handle start", zap.Error(err))
		}
		return
	}

	err := dialog.next(u)
	if err != nil {
		m.L.Error("handle request", zap.Error(err))
	}
}

func (m *MessageHandler) handleCallbackQuery(u *Update) error {
	d := u.CallbackQuery.Data
	qslice := strings.Split(d, "|")

	if len(qslice) != 2 {
		return fmt.Errorf("invalid callbackquery")
	}

	switch qslice[0] {
	case cmdMyHelp: // delete help
		uid, err := uuid.Parse(qslice[1])
		if err != nil {
			return fmt.Errorf("parse uuid: %w", err)
		}

		err = m.Service.DeleteHelp(u.ctx, uid)
		if err != nil {
			return fmt.Errorf("parse uuid: %w", err)
		}

		msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%s.\n\n%s", m.Localize.Translate(deleteHelpSuccessTr, UALang), m.Localize.Translate(navigationHintTr, UALang)))
		msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
		_, err = m.Api.Send(msg)
		return err

	case cmdMySubscriptions:
		sid, err := uuid.Parse(qslice[1])
		if err != nil {
			return fmt.Errorf("parse uuid: %w", err)
		}

		ok, err := m.Service.SubscriptionExists(u.ctx, sid)
		if err != nil {
			return fmt.Errorf("check subscription exists: %w", err)
		}

		if !ok {
			msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%s.\n\n%s", m.Localize.Translate(errorSubscriptionDoesNotExistTr, UALang), m.Localize.Translate(navigationHintTr, UALang)))
			msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
			_, err = m.Api.Send(msg)
			return err
		}

		err = m.Service.DeleteSubscription(u.ctx, sid)
		if err != nil {
			return fmt.Errorf("parse uuid: %w", err)
		}

		msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%s.\n\n%s", m.Localize.Translate(deleteSubscriptionSuccessTr, UALang), m.Localize.Translate(navigationHintTr, UALang)))
		msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
		_, err = m.Api.Send(msg)
		return err

	case cqHelpsBySubscription:
		sid, err := uuid.Parse(qslice[1])
		if err != nil {
			return fmt.Errorf("parse subscription id: %w", err)
		}

		ok, err := m.Service.SubscriptionExists(u.ctx, sid)
		if err != nil {
			return fmt.Errorf("check subscription exists: %w", err)
		}

		if !ok {
			msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%s.\n\n%s", m.Localize.Translate(errorSubscriptionDoesNotExistTr, UALang), m.Localize.Translate(navigationHintTr, UALang)))
			msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
			_, err = m.Api.Send(msg)
			return err
		}

		helps, err := m.Service.HelpsBySubscription(u.ctx, sid)
		if err != nil {
			return err
		}

		if len(helps) == 0 {
			msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%s.\n\n%s", m.Localize.Translate(seekerHelpsEmptyTr, UALang), m.Localize.Translate(navigationHintTr, UALang)))
			msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
			_, err = m.Api.Send(msg)
			return err
		}

		for _, help := range helps {
			var b strings.Builder
			b.WriteString(fmt.Sprintf("%s %s\n", emojiLocation, help.Locality))
			b.WriteString(fmt.Sprintf("%s %s\n", emojiTime, m.Localize.FormatDateTime(help.CreatedAt, UALang)))
			for _, c := range help.Categories {
				b.WriteString(fmt.Sprintf("%s %s\n", emojiItem, c))
			}
			b.WriteString(fmt.Sprintf("%s\n\n", help.Description))
			msg := tg.NewMessage(u.chatID(), b.String())
			msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
			_, err = m.Api.Send(msg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *MessageHandler) handleCmdStart(u *Update) error {
	activity, err := m.Service.GetActivityStats(u.ctx)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s\n", m.Localize.Translate(cmdStartActivityHeaderTr, UALang)))
	b.WriteString(fmt.Sprintf("%s %d\n", m.Localize.Translate(cmdStartActivityHelpsTr, UALang), activity.ActiveHelpsCount))
	b.WriteString(fmt.Sprintf("%s %d\n\n", m.Localize.Translate(cmdStartActivitySubscriptionsTr, UALang), activity.ActiveSubsCount))
	b.WriteString(m.Localize.Translate(userRoleRequestTr, UALang))

	msg := tg.NewMessage(u.chatID(), b.String())
	msg.ReplyMarkup = tg.ReplyKeyboardMarkup{
		OneTimeKeyboard: false,
		ResizeKeyboard:  true,
		Keyboard: [][]tg.KeyboardButton{
			{tg.KeyboardButton{Text: m.Localize.Translate(btnOptionRoleSeekerTr, UALang)}},
			{tg.KeyboardButton{Text: m.Localize.Translate(btnOptionUserVolunteerTr, UALang)}},
			{tg.KeyboardButton{Text: m.Localize.Translate(btnOptionCancelTr, UALang)}},
		},
	}

	_, err = m.Api.Send(msg)
	if err != nil {
		return err
	}

	m.dialogs.set(&dialog{next: m.handleUserRoleReply}, u.chatID())
	return nil
}

func (m *MessageHandler) handleUserRoleReply(u *Update) error {
	switch u.Message.Text {
	case m.Localize.Translate(btnOptionRoleSeekerTr, UALang):
		return m.handleSeekerUserRoleReply(u)
	case m.Localize.Translate(btnOptionUserVolunteerTr, UALang):
		return m.handleVolunteerUserRoleReply(u)
	default:
		_, err := m.Api.Send(tg.NewMessage(u.chatID(), m.Localize.Translate(errorChooseOptionTr, UALang)))
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MessageHandler) handleCmdSupport(u *Update) error {
	msg := tg.NewMessage(u.chatID(), fmt.Sprintf("%s\n\n%s", m.Localize.Translate(cmdSupportTr, UALang), m.Localize.Translate(navigationHintTr, UALang)))
	msg.ReplyMarkup = tg.ReplyKeyboardHide{HideKeyboard: true}
	_, err := m.Api.Send(msg)
	return err
}
