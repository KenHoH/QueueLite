package app

import (
	businessdomain "QueueLite/internal/business/domain"
	counterdomain "QueueLite/internal/counter/domain"
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

type CustomerQueueStatusData struct {
	queue    *domain.Queue
	business *businessdomain.Business
	counters []counterdomain.Counter
	queues   []domain.Queue
	waiting  []QueueSnapshotItem
}
