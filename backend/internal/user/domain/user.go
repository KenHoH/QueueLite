package domain

import (
	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	Password    string
	Username    string
	PhoneNumber string
	Email       *string
}
