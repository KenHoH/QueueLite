package app

import (
	"time"

	businessdomain "QueueLite/internal/business/domain"
	"QueueLite/internal/queue/domain"

	"github.com/google/uuid"
)

// QueueSnapshotItem represents a queue entry in the application-level waiting snapshot.
type QueueSnapshotItem struct {
	QueueID   string
	QueueName string
}

// CustomerCounterStatus describes the queue currently assigned to a counter.
type CustomerCounterStatus struct {
	ID               uuid.UUID
	Name             string
	CurrentQueueID   *uuid.UUID
	CurrentQueueName *string
	State            string
}

// CurrentlyServingQueue describes an active queue and its assigned counter.
type CurrentlyServingQueue struct {
	CounterID   uuid.UUID
	CounterName string
	QueueID     uuid.UUID
	QueueName   string
	State       domain.QueueState
}

// CustomerQueuePosition describes a customer's place in the waiting queue.
type CustomerQueuePosition struct {
	Position  int
	Ahead     int
	QueueName string
}

// CustomerQueueStatus is the application read model assembled for queue status.
type CustomerQueueStatus struct {
	Queue                domain.Queue
	Business             businessdomain.Business
	Counters             []CustomerCounterStatus
	TotalCounters        int
	ActiveCounters       int
	CurrentlyServing     []CurrentlyServingQueue
	NextQueue            *QueueSnapshotItem
	CustomerPosition     *CustomerQueuePosition
	TotalWaiting         int
	TotalActiveQueues    int
	EstimatedWaitMinutes *int
	UpdatedAt            time.Time
}
