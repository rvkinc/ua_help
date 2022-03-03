package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/rvkinc/uasocial/internal/storage"
)

type Interface interface {
}

type PushMessage struct {
	Recipients []uuid.UUID
	Message    Message
}

type Message struct {
	Text string
}

type Request struct {
	ID          uuid.UUID
	CreatorID   uuid.UUID
	CategoryID  uuid.UUID
	LocalityID  int
	Description string
	Resolved    bool
	CreatedAt   time.Time
}

type CreateUser struct {
	TgID   int64
	ChatID int64
	Name   string
}

type User struct {
	ID     uuid.UUID
	TgID   int64
	ChatID int64
	Name   string
}

type CreateHelp struct {
	CreatorID  uuid.UUID
	CategoryID uuid.UUID
	LocalityID int
}

type UserHelp struct {
	ID         uuid.UUID
	CreatorID  uuid.UUID
	CategoryID uuid.UUID
	LocalityID int
}

type locality string

type Category string

type UserRequest struct {
	Category    Category
	Phone       string
	Locality    locality
	Description string
	CreatedAt   time.Time
}

type NewRequest struct {
	CreatorID    uuid.UUID
	CategoryID   uuid.UUID
	LocalityID   int
	LocalityType string
	Phone        string
	Description  string
}

type HelpMessage struct {
	ChatID int64
}

type RequestMessage struct {
}

type Locality struct {
	ID         int
	Type       string
	Name       string
	RegionName string
}

type language string

const (
	UA language = "UA"
	EN language = "EN"
	RU language = "RU"
)

// Service is a service implementation.
type Service struct {
	storage          storage.Interface
	language         language
	requestMessageCh chan RequestMessage
	helpMessageCh    chan []HelpMessage
}

// NewService returns new service implementation.
func NewService(storage storage.Interface, language language) *Service {
	return &Service{
		storage:          storage,
		language:         language,
		requestMessageCh: make(chan RequestMessage, 100),
		helpMessageCh:    make(chan []HelpMessage, 100),
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
	ls, err := s.storage.SelectLocalities(ctx, input)
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

// NewHelp creates new help.
func (s *Service) NewHelp(ctx context.Context, request CreateHelp) error {
	_, err := s.storage.InsertHelp(ctx, &storage.Help{
		CreatorID:  request.CreatorID,
		CategoryID: request.CategoryID,
		LocalityID: request.LocalityID,
	})
	if err != nil {
		return err
	}
	return nil
}

// UserHelps returns helps of specific userID.
func (s *Service) UserHelps(ctx context.Context, userID uuid.UUID) ([]UserHelp, error) {
	hs, err := s.storage.SelectHelpsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	helps := make([]UserHelp, 0, len(hs))
	for _, help := range hs {
		helps = append(helps, UserHelp{
			CategoryID: help.CategoryID,
		})
	}
	return helps, nil
}

// DeleteHelp deletes specific help by helpID.
func (s *Service) DeleteHelp(ctx context.Context, helpID uuid.UUID) error {
	return s.storage.DeleteHelp(ctx, helpID)
}

// NewRequest creates request.
func (s *Service) NewRequest(ctx context.Context, request NewRequest) error {
	_, err := s.storage.InsertRequest(ctx, &storage.Request{
		CreatorID:  request.CreatorID,
		CategoryID: request.CategoryID,
		LocalityID: request.LocalityID,
		Phone: sql.NullString{
			String: request.Phone,
			Valid:  true,
		},
		Description: request.Description,
	})
	if err != nil {
		return err
	}

	var (
		hs []*storage.Help
	)

	switch request.LocalityType {
	case "VILLAGE", "URBAN":
		hs, err = s.storage.SelectHelpsByLocalityAndCategoryForVillage(ctx, request.LocalityID, request.CategoryID)
	default:
		hs, err = s.storage.SelectHelpsByLocalityAndCategory(ctx, request.LocalityID, request.CategoryID)
	}

	if err != nil {
		return err
	}

	helpMessages := make([]HelpMessage, 0, len(hs))

	for _, help := range hs {
		helper, err := s.storage.UserByID(ctx, help.CreatorID)
		if err != nil {
			return err
		}
		helpMessages = append(helpMessages, HelpMessage{
			ChatID: helper.ChatID,
		})
	}

	go s.sendHelpMessages(helpMessages)

	return nil
}

func (s *Service) sendHelpMessages(helpMessages []HelpMessage) {
	s.helpMessageCh <- helpMessages
}

// UserRequests returns user's requests.
func (s *Service) UserRequests(ctx context.Context, userID uuid.UUID) ([]UserRequest, error) {
	//
	return nil, nil
}

// UserRequest returns user's request.
func (s *Service) UserRequest(ctx context.Context, requestID uuid.UUID) (UserRequest, error) {
	//
	return UserRequest{}, nil
}

// DeleteRequest deletes specific request by requestID.
func (s *Service) DeleteRequest(ctx context.Context, requestID uuid.UUID) error {
	//
	return nil
}
