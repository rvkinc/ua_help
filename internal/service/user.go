package service

import "time"

type User struct {
	ID         string
	TelegramID int
	Name       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
