package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type localityID = int32

type Service interface {
	NewRequest(ctx context.Context, rq *Request) (*Request, error)
	LocalityRequests(lid localityID) ([]*Request, error)
	CloseRequest(rid uuid.UUID) error

	Push() <-chan PushMessage
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
	LocalityID  localityID
	Phone       string
	Description string
	Resolved    bool
	CreatedAt   time.Time
}
