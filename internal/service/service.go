package service

import (
	"context"
	"github.com/google/uuid"
	"time"
)

type localityID = int32

type Service interface {
	NewRequest(ctx context.Context, rq *Request) (*Request, error)
	// resovle request...
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
	Description string
	Resolved    bool
	CreatedAt   time.Time
}
