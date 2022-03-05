package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rvkinc/uasocial/internal/storage"
)

var (
	tenDaysDuration = time.Hour * 24 * 10
	tenDaysDate     = time.Now().AddDate(0, 0, -10)
)

type (
	CreateUser struct {
		TgID   int64
		ChatID int64
		Name   string
	}

	User struct {
		ID     uuid.UUID
		TgID   int64
		ChatID int64
		Name   string
	}

	CreateSubscription struct {
		CreatorID  uuid.UUID
		CategoryID uuid.UUID
		LocalityID int
	}

	UserHelp struct {
		ID          uuid.UUID
		CreatorID   uuid.UUID
		Categories  []string
		Locality    string
		Description string
		CreatedAt   time.Time
	}

	UserSubscription struct {
		ID        uuid.UUID
		CreatorID uuid.UUID
		Category  string
		Locality  string
		CreatedAt time.Time
	}

	NewHelp struct {
		CreatorID   uuid.UUID
		CategoryIDs []uuid.UUID
		LocalityID  int
		Description string
	}

	SubscriptionMessage struct {
		ChatID int64
		UserHelp
	}

	Locality struct {
		ID         int
		Type       string
		Name       string
		RegionName string
	}
)

// Service is a service implementation.
type Service struct {
	storage                storage.Interface
	expiredHelpsCh         chan []UserHelp
	subscriptionsMessageCh chan []SubscriptionMessage
}

// NewService returns new service implementation.
func NewService(storage storage.Interface) *Service {
	s := &Service{
		storage:                storage,
		expiredHelpsCh:         make(chan []UserHelp),
		subscriptionsMessageCh: make(chan []SubscriptionMessage, 100),
	}

	go s.handleExpiredHelps()

	return s
}

func (s *Service) handleExpiredHelps() {
	ticker := time.NewTicker(tenDaysDuration).C
	for range ticker {
		helps, err := s.expiredHelps(context.Background(), tenDaysDate)
		if err != nil {
			// log here
			continue
		}
		s.expiredHelpsCh <- helps
	}
}

// NewUser creates new user or returns an existing.
func (s *Service) NewUser(ctx context.Context, user CreateUser) (User, error) {
	u, err := s.storage.UpsertUser(ctx, &storage.User{
		TgID:   user.TgID,
		ChatID: user.ChatID,
		Name:   user.Name,
	})
	if err != nil {
		return User{}, err
	}
	return User{
		ID:     u.ID,
		TgID:   u.TgID,
		ChatID: u.ChatID,
		Name:   u.Name,
	}, nil
}

// AutocompleteLocality returns matched localities.
func (s *Service) AutocompleteLocality(ctx context.Context, input string) ([]Locality, error) {
	ls, err := s.storage.SelectLocalityRegions(ctx, input)
	if err != nil {
		return nil, err
	}
	localities := make([]Locality, 0, len(ls))
	for _, locality := range ls {
		localities = append(localities, Locality{
			ID:         locality.ID,
			Name:       locality.Name,
			Type:       locality.Type,
			RegionName: locality.RegionName,
		})
	}
	return localities, nil
}

// NewSubscription creates new subscription.
func (s *Service) NewSubscription(ctx context.Context, subscription CreateSubscription) error {
	return s.storage.InsertSubscription(ctx, &storage.SubscriptionInsert{
		CreatorID:  subscription.CreatorID,
		CategoryID: subscription.CategoryID,
		LocalityID: subscription.LocalityID,
	})
}

// UserSubscriptions returns subscription of specific userID.
func (s *Service) UserSubscriptions(ctx context.Context, userID uuid.UUID) ([]UserSubscription, error) {
	ss, err := s.storage.SelectSubscriptionsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	subscriptions := make([]UserSubscription, 0, len(ss))
	for _, subscription := range ss {
		s := UserSubscription{
			ID:        subscription.ID,
			CreatorID: subscription.CreatorID,
		}
		s.localize(subscription)
		subscriptions = append(subscriptions, s)
	}
	return subscriptions, nil
}

func (h *UserHelp) localize(help *storage.Help) {
	categories := make([]string, 0, len(help.Categories))
	switch help.Language {
	case "UA":
		h.Locality = help.LocalityPublicNameUA
		for _, category := range help.Categories {
			categories = append(categories, category.NameUA)
		}
	case "RU":
		h.Locality = help.LocalityPublicNameRU
		for _, category := range help.Categories {
			categories = append(categories, category.NameRU)
		}
	case "EN":
		for _, category := range help.Categories {
			categories = append(categories, category.NameEN)
		}
		h.Locality = help.LocalityPublicNameEN
	}
	h.Categories = categories
}

func (us *UserSubscription) localize(subscription *storage.SubscriptionValue) {
	switch subscription.Language {
	case "UA":
		us.Category, us.Locality = subscription.CategoryNameUA, subscription.LocalityPublicNameUA
	case "RU":
		us.Category, us.Locality = subscription.CategoryNameRU, subscription.LocalityPublicNameRU
	case "EN":
		us.Category, us.Locality = subscription.CategoryNameEN, subscription.LocalityPublicNameEN
	}
}

// DeleteHelp deletes specific help by helpID.
func (s *Service) DeleteHelp(ctx context.Context, helpID uuid.UUID) error {
	return s.storage.DeleteHelp(ctx, helpID)
}

// DeleteSubscription deletes specific subscription by helpID.
func (s *Service) DeleteSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	return s.storage.DeleteSubscription(ctx, subscriptionID)
}

// NewHelp creates new help.
func (s *Service) NewHelp(ctx context.Context, help NewHelp) error {
	helpID, err := s.storage.InsertHelp(ctx, &storage.HelpInsert{
		CreatorID:   help.CreatorID,
		CategoryIDs: help.CategoryIDs,
		LocalityID:  help.LocalityID,
		Description: help.Description,
	})
	if err != nil {
		return err
	}

	helpValue, err := s.storage.SelectHelpByID(ctx, helpID)
	if err != nil {
		return err
	}

	subscriptions, err := s.storage.SelectSubscriptionsByLocalityCategories(ctx, help.LocalityID, help.CategoryIDs)
	if err != nil {
		return err
	}

	subscriptionMessages := make([]SubscriptionMessage, 0, len(subscriptions))

	for _, subscription := range subscriptions {
		u := UserHelp{
			ID:          helpValue.ID,
			CreatorID:   helpValue.CreatorID,
			Description: helpValue.Description,
			CreatedAt:   helpValue.CreatedAt,
		}
		u.localize(helpValue)
		subscriptionMessages = append(subscriptionMessages, SubscriptionMessage{
			UserHelp: u,
			ChatID:   subscription.ChatID,
		})
	}

	go s.notifySubscriptions(subscriptionMessages)

	return nil
}

func (s *Service) notifySubscriptions(subscriptionMessages []SubscriptionMessage) {
	s.subscriptionsMessageCh <- subscriptionMessages
}

// UserHelps returns user's helps.
func (s *Service) UserHelps(ctx context.Context, userID uuid.UUID) ([]UserHelp, error) {
	hs, err := s.storage.SelectHelpsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	helps := make([]UserHelp, 0, len(hs))
	for _, help := range hs {
		h := UserHelp{
			ID:          help.ID,
			CreatorID:   help.CreatorID,
			Description: help.Description,
			CreatedAt:   help.CreatedAt,
		}
		h.localize(help)
		helps = append(helps, h)
	}
	return helps, nil
}

func (s *Service) expiredHelps(ctx context.Context, after time.Time) ([]UserHelp, error) {
	hs, err := s.storage.SelectExpiredHelps(ctx, after)
	if err != nil {
		return nil, err
	}
	helps := make([]UserHelp, 0, len(hs))
	for _, help := range hs {
		h := UserHelp{
			ID:          help.ID,
			CreatorID:   help.CreatorID,
			Description: help.Description,
			CreatedAt:   help.CreatedAt,
		}
		h.localize(help)
		helps = append(helps, h)
	}
	return helps, nil
}

// KeepHelp keeps help.
func (s *Service) KeepHelp(ctx context.Context, helpID uuid.UUID) error {
	return s.storage.KeepHelp(ctx, helpID)
}
