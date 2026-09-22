package domain

import (
	"time"

	"github.com/google/uuid"
)

type QueueState string

const (
	QueueStateWaiting   QueueState = "waiting"
	QueueStateCalled    QueueState = "called"
	QueueStateProcess   QueueState = "process"
	QueueStateCanceled  QueueState = "canceled"
	QueueStateCompleted QueueState = "done"
)

type Queue struct {
	ID         uuid.UUID
	BusinessID uuid.UUID
	UserID     uuid.UUID
	CounterID  *uuid.UUID
	Name       string
	State      QueueState
	Priority   bool
	StartTime  *time.Time
	EndTime    *time.Time
}
