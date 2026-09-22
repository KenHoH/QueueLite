package domain

import (
	"time"

	"github.com/google/uuid"
)

type Business struct {
	ID          uuid.UUID
	Name        string
	Location    string
	Description *string
	Operational bool
	OpenTime    time.Time
	CloseTime   time.Time
	CreatedAt   time.Time
	Email       string
	PhoneNumber string
}

type UpdateBusiness struct {
	Name        *string
	Location    *string
	Description *string
	Operational *bool
	OpenTime    *time.Time
	CloseTime   *time.Time
	Email       *string
	PhoneNumber *string
}

type BusinessCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}
