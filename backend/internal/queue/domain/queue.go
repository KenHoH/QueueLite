package domain

import (
	"time"

	"github.com/google/uuid"
)

type QueueState string

const (
	QueueStateWaiting    QueueState = "waiting"
	QueueStateCalled     QueueState = "called"
	QueueStateProcessing QueueState = "processing"
	QueueStateCancelled  QueueState = "cancelled"
	QueueStateSkipped    QueueState = "skipped"
	QueueStateCompleted  QueueState = "done"
)

type Queue struct {
	ID         uuid.UUID
	BusinessID uuid.UUID
	UserID     *uuid.UUID

	CalledByCounterID *uuid.UUID

	Name string

	State    QueueState
	Priority bool

	CalledAt     *time.Time
	ProcessingAt *time.Time
	DoneAt       *time.Time
	CancelledAt  *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type PublicQueueSummary struct {
	CurrentQueueName *string
	TotalWaiting     int64
	NextQueueName    *string
}
