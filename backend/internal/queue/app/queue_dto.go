package app

import (
	"QueueLite/internal/queue/domain"

	"github.com/google/uuid"
)

type RegisterCustomerQueueInput struct {
	BusinessID  uuid.UUID
	UserID      *uuid.UUID
	Username    string
	PhoneNumber string
}

type RegisterCustomerQueueResult struct {
	Queue       *domain.Queue
	GuestID     *uuid.UUID
	Username    string
	PhoneNumber string
}
