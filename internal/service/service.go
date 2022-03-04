package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/rvkinc/uasocial/internal/storage"
)

var (
	tenDaysDuration = time.Hour * 24 * 10
	tenDaysDate     = time.Now().AddDate(0, 0, -10)
)

type (
	Message struct {
		Text string
	}

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

	CreateHelp struct {
		CreatorID  uuid.UUID
		CategoryID uuid.UUID
		LocalityID int
	}

	UserHelp struct {
		ID        uuid.UUID
		CreatorID uuid.UUID
		Category  string
		Locality  string
	}

	UserRequest struct {
		ID          uuid.UUID
		Category    string
		TgID        string
		Phone       string
		Locality    string
		Description string
		CreatedAt   time.Time
	}

	NewRequest struct {
		CreatorID    uuid.UUID
		CategoryID   uuid.UUID
		TgID         int64
		LocalityID   int
		LocalityType string
		Phone        string
		Description  string
	}

	HelpMessage struct {
		ChatID int64
		UserRequest
	}

	Locality struct {
		ID         int
		Type       string
		Name       string
		RegionName string
	}

	language string
)

const (
	UA language = "UA"
	EN language = "EN"
	RU language = "RU"
)

// Service is a service implementation.
type Service struct {
	storage           storage.Interface
	language          language
	requestsExpiredCh chan []UserRequest
	helpMessageCh     chan []HelpMessage
}

// NewService returns new service implementation.
func NewService(storage storage.Interface, language language) *Service {
	s := &Service{
		storage:           storage,
		language:          language,
		requestsExpiredCh: make(chan []UserRequest),
		helpMessageCh:     make(chan []HelpMessage, 100),
	}

	go s.checkExpiredRequests()

	return s
}

func (s *Service) checkExpiredRequests() {
	ticker := time.NewTicker(tenDaysDuration).C
	for range ticker {
		requests, err := s.expiredRequests(context.Background(), tenDaysDate)
		if err != nil {
			// log here
			continue
		}
		s.requestsExpiredCh <- requests
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
	return s.storage.InsertHelp(ctx, &storage.HelpScan{
		CreatorID:  request.CreatorID,
		CategoryID: request.CategoryID,
		LocalityID: request.LocalityID,
	})
}

// UserHelps returns helps of specific userID.
func (s *Service) UserHelps(ctx context.Context, userID uuid.UUID) ([]UserHelp, error) {
	hs, err := s.storage.SelectHelpsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	helps := make([]UserHelp, 0, len(hs))
	for _, help := range hs {
		h := UserHelp{
			ID:        help.ID,
			CreatorID: help.CreatorID,
		}
		h.localize(help)
		helps = append(helps, h)
	}
	return helps, nil
}

func (h *UserHelp) localize(help *storage.HelpValue) {
	switch help.Language {
	case "UA":
		h.Category, h.Locality = help.CategoryNameUA, help.LocalityPublicNameUA
	case "RU":
		h.Category, h.Locality = help.CategoryNameRU, help.LocalityPublicNameRU
	case "EN":
		h.Category, h.Locality = help.CategoryNameEN, help.LocalityPublicNameEN
	}
}

func (u *UserRequest) localize(request *storage.RequestValue) {
	switch request.Language {
	case "UA":
		u.Category, u.Locality = request.CategoryNameUA, request.LocalityPublicNameUA
	case "RU":
		u.Category, u.Locality = request.CategoryNameRU, request.LocalityPublicNameRU
	case "EN":
		u.Category, u.Locality = request.CategoryNameEN, request.LocalityPublicNameEN
	}
}

// DeleteHelp deletes specific help by helpID.
func (s *Service) DeleteHelp(ctx context.Context, helpID uuid.UUID) error {
	return s.storage.DeleteHelp(ctx, helpID)
}

// NewRequest creates request.
func (s *Service) NewRequest(ctx context.Context, request NewRequest) error {
	var isValid bool

	if request.Phone != "" {
		isValid = true
	}

	requestValue, err := s.storage.InsertRequest(ctx, &storage.RequestScan{
		CreatorID:  request.CreatorID,
		CategoryID: request.CategoryID,
		LocalityID: request.LocalityID,
		Phone: sql.NullString{
			String: request.Phone,
			Valid:  isValid,
		},
		Description: request.Description,
	})
	if err != nil {
		return err
	}

	var (
		hs []*storage.User
	)

	switch request.LocalityType {
	case "VILLAGE", "URBAN":
		hs, err = s.storage.SelectHelpsByLocalityAndCategoryForVillage(ctx, request.LocalityID, request.CategoryID)
	default:
		hs, err = s.storage.SelectHelpsByLocalityAndCategoryForCity(ctx, request.LocalityID, request.CategoryID)
	}

	if err != nil {
		return err
	}

	helpMessages := make([]HelpMessage, 0, len(hs))

	for _, help := range hs {
		u := UserRequest{
			ID:          requestValue.ID,
			TgID:        requestValue.TgID,
			Phone:       requestValue.Phone.String,
			Description: requestValue.Description,
			CreatedAt:   requestValue.CreatedAt,
		}
		u.localize(requestValue)
		helpMessages = append(helpMessages, HelpMessage{
			UserRequest: u,
			ChatID:      help.ChatID,
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
	rs, err := s.storage.SelectRequestsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	requests := make([]UserRequest, 0, len(rs))
	for _, request := range rs {
		u := UserRequest{
			ID:          request.ID,
			TgID:        request.TgID,
			Phone:       request.Phone.String,
			Description: request.Description,
			CreatedAt:   request.CreatedAt,
		}
		u.localize(request)
		requests = append(requests, u)
	}

	return requests, nil
}

// DeleteRequest deletes specific request by requestID.
func (s *Service) DeleteRequest(ctx context.Context, requestID uuid.UUID) error {
	return s.storage.ResolveRequest(ctx, requestID)
}

func (s *Service) expiredRequests(ctx context.Context, after time.Time) ([]UserRequest, error) {
	rs, err := s.storage.ExpiredRequests(ctx, after)
	if err != nil {
		return nil, err
	}
	requests := make([]UserRequest, 0, len(rs))
	for _, request := range rs {
		u := UserRequest{
			ID:          request.ID,
			TgID:        request.TgID,
			Phone:       request.Phone.String,
			Description: request.Description,
			CreatedAt:   request.CreatedAt,
		}
		u.localize(request)
		requests = append(requests, u)
	}
	return requests, nil
}

func (s *Service) KeepRequest(ctx context.Context, requestID uuid.UUID) error {
	return s.storage.KeepRequest(ctx, requestID)
}
