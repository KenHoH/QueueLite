// Package testutil supplies in-memory repository boundaries for queue contract tests.
package testutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"QueueLite/internal/apperror"
	businessdomain "QueueLite/internal/business/domain"
	queueapp "QueueLite/internal/queue/app"
	queuedomain "QueueLite/internal/queue/domain"
	userapp "QueueLite/internal/user/app"
	userdomain "QueueLite/internal/user/domain"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type QueueFixture struct {
	queueapp.QueueRepo
	userapp.UserRepo
	Business      businessdomain.Business
	User          userdomain.User
	BusinessError error
	mu            sync.Mutex
	Queues        map[uuid.UUID]queuedomain.Queue
	Capacity      int
	Redis         *redis.Client
	Server        *miniredis.Miniredis
	Service       *queueapp.QueueService
}

func NewQueueFixture(t *testing.T) *QueueFixture {
	t.Helper()
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	f := &QueueFixture{Business: businessdomain.Business{ID: uuid.New(), Operational: true}, User: userdomain.User{ID: uuid.New(), Username: "Stored name", PhoneNumber: "081234567890"}, Queues: make(map[uuid.UUID]queuedomain.Queue), Capacity: 100, Redis: rdb, Server: server}
	f.Service = queueapp.NewQueueService(f, f, rdb, f)
	return f
}

func (f *QueueFixture) GetBusiness(_ context.Context, id uuid.UUID) (*businessdomain.Business, error) {
	if f.BusinessError != nil {
		return nil, f.BusinessError
	}
	if id != f.Business.ID {
		return nil, businessdomain.ErrBusinessNotFound
	}
	return &f.Business, nil
}
func (f *QueueFixture) GetUser(_ context.Context, id uuid.UUID) (*userdomain.User, error) {
	if id != f.User.ID {
		return nil, userapp.ErrUserNotFound
	}
	return &f.User, nil
}
func (f *QueueFixture) GetDailyQueueNumberFloor(_ context.Context, _ uuid.UUID, _ time.Time) (int64, error) {
	return 0, nil
}
func (f *QueueFixture) GetActiveQueueByUserAndBusiness(_ context.Context, owner, business uuid.UUID) (*queuedomain.Queue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, q := range f.Queues {
		if q.BusinessID == business && q.UserID != nil && *q.UserID == owner && !q.IsTerminal() {
			return &q, nil
		}
	}
	return nil, nil
}
func (f *QueueFixture) GetQueue(_ context.Context, id uuid.UUID) (*queuedomain.Queue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	q, ok := f.Queues[id]
	if !ok {
		return nil, queueapp.ErrQueueNotFound
	}
	return &q, nil
}

func (f *QueueFixture) GetActiveQueuesByUser(_ context.Context, userID uuid.UUID) ([]queuedomain.Queue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	queues := make([]queuedomain.Queue, 0)
	for _, q := range f.Queues {
		if q.UserID != nil && *q.UserID == userID && !q.IsTerminal() {
			queues = append(queues, q)
		}
	}
	return queues, nil
}
func (f *QueueFixture) UpdateQueue(_ context.Context, q *queuedomain.Queue) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.Queues[q.ID]; !ok {
		return queueapp.ErrQueueNotFound
	}
	f.Queues[q.ID] = *q
	return nil
}

func (f *QueueFixture) UpdateQueueIfState(_ context.Context, q *queuedomain.Queue, expected queuedomain.QueueState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.Queues[q.ID]
	if !ok || stored.State != expected {
		return apperror.New(apperror.KindConflict, "QUEUE_STATE_CHANGED", "queue state changed")
	}
	f.Queues[q.ID] = *q
	return nil
}
func (f *QueueFixture) DeleteQueue(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.Queues[id]; !ok {
		return queueapp.ErrQueueNotFound
	}
	delete(f.Queues, id)
	return nil
}
func (f *QueueFixture) Persist(q queuedomain.Queue) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Queues[q.ID] = q
}
func (f *QueueFixture) DecreaseBusinessCapacity(_ context.Context, _ uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Capacity <= 0 {
		return apperror.New(apperror.KindConflict, "BUSINESS_QUEUE_FULL", "full")
	}
	f.Capacity--
	return nil
}
func (f *QueueFixture) IncreaseBusinessCapacity(_ context.Context, _ uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Capacity++
	return nil
}
func (f *QueueFixture) CheckBusinessQueueQuota(_ context.Context, _ uuid.UUID) error { return nil }
func (f *QueueFixture) CheckUserPrioritySlot(_ context.Context, _ uuid.UUID) error   { return nil }
