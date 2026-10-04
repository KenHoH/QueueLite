package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// BusinessMembership keeps account membership separate from public business data.
type BusinessMembership struct {
	Business Business
	Role     string
}

type BusinessMember struct {
	UserID   uuid.UUID `json:"userId"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
}

type MembershipUser struct {
	ID       uuid.UUID
	Username string
}

var (
	ErrBusinessNotFound       = errors.New("business not found")
	ErrMembershipUserNotFound = errors.New("membership user not found")
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
