package service

import "time"

type Help struct {
	ID         string
	CreatorID  string
	CategoryID string
	LocalityID int
	CreatedAt  time.Time
	DeletedAt  time.Time
}
