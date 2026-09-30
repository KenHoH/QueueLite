package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	cache "QueueLite/internal/adapter/redis"
	counterapp "QueueLite/internal/counter/app"
	counterdomain "QueueLite/internal/counter/domain"
	queueapp "QueueLite/internal/queue/app"
	queuedomain "QueueLite/internal/queue/domain"
	"QueueLite/internal/testutil"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type failingCallQueueRepo struct {
	counterapp.CounterQueueRepo
	readError   error
	updateError error
	cancelRead  context.CancelFunc
}

func (r *failingCallQueueRepo) GetQueue(ctx context.Context, id uuid.UUID) (*queuedomain.Queue, error) {
	if r.cancelRead != nil {
		r.cancelRead()
		return nil, queueapp.ErrQueueNotFound
	}
	if r.readError != nil {
		return nil, r.readError
	}
	return r.CounterQueueRepo.GetQueue(ctx, id)
}

func (r *failingCallQueueRepo) UpdateQueue(ctx context.Context, q *queuedomain.Queue) error {
	if r.updateError != nil {
		return r.updateError
	}
	return r.CounterQueueRepo.UpdateQueue(ctx, q)
}

type failingAssignmentRepo struct {
	*counterRepo
	assignmentError error
}

func (r *failingAssignmentRepo) UpdateCounterCustomer(ctx context.Context, counterID uuid.UUID, queueID *uuid.UUID) error {
	if r.assignmentError != nil {
		return r.assignmentError
	}
	return r.counterRepo.UpdateCounterCustomer(ctx, counterID, queueID)
}

func TestCallNextRestoresFailedSelectionToOriginalKey(t *testing.T) {
	for _, kind := range []string{"priority", "normal", "legacy"} {
		for _, failure := range []string{"invalid-id", "read", "cancelled-persistence-wait", "queue-update", "counter-assignment"} {
			t.Run(kind+"/"+failure, func(t *testing.T) {
				f := testutil.NewQueueFixture(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				owner := uuid.New()
				q := queuedomain.Queue{ID: uuid.New(), BusinessID: f.Business.ID, UserID: &owner, Name: "A001", State: queuedomain.QueueStateWaiting, Priority: kind == "priority", CreatedAt: time.Now().UTC()}
				f.Persist(q)
				businessID := q.BusinessID.String()
				keys := map[string]string{"priority": cache.PriorityWaitingQueueKey(businessID), "normal": cache.NormalWaitingQueueKey(businessID), "legacy": cache.LegacyWaitingQueueKey(businessID)}
				key := keys[kind]
				member := q.ID.String()
				if failure == "invalid-id" {
					member = "invalid-uuid"
				}
				if err := f.Redis.ZAdd(ctx, key, redis.Z{Member: member, Score: cache.QueueScore(q)}).Err(); err != nil {
					t.Fatal(err)
				}
				counterID := uuid.New()
				r := &failingAssignmentRepo{counterRepo: &counterRepo{counter: counterdomain.Counter{ID: counterID, BusinessID: q.BusinessID}}}
				qr := &failingCallQueueRepo{CounterQueueRepo: f}
				injected := errors.New("injected repository failure")
				switch failure {
				case "read":
					qr.readError = injected
				case "cancelled-persistence-wait":
					qr.cancelRead = cancel
				case "queue-update":
					qr.updateError = injected
				case "counter-assignment":
					r.assignmentError = injected
				}
				s := counterapp.NewCounterService(r, qr, nil, f.Redis)
				if result, err := s.CallNextQueue(ctx, counterID, q.BusinessID); err == nil || result != nil {
					t.Fatalf("failure not propagated: %#v %v", result, err)
				} else if failure == "cancelled-persistence-wait" && !errors.Is(err, context.Canceled) {
					t.Fatalf("lost cancellation cause: %v", err)
				} else if failure != "invalid-id" && failure != "cancelled-persistence-wait" && !errors.Is(err, injected) {
					t.Fatalf("lost repository cause: %v", err)
				}
				for _, candidate := range keys {
					items, err := f.Redis.ZRangeWithScores(context.Background(), candidate, 0, -1).Result()
					if err != nil {
						t.Fatal(err)
					}
					if candidate == key {
						if len(items) != 1 || items[0].Member != member || items[0].Score != cache.QueueScore(q) {
							t.Fatalf("source key/UUID/score not restored: %#v", items)
						}
					} else if len(items) != 0 {
						t.Fatalf("restored to wrong key %s: %#v", candidate, items)
					}
				}
				stored, err := f.GetQueue(context.Background(), q.ID)
				if err != nil || stored.State != queuedomain.QueueStateWaiting || stored.CalledByCounterID != nil || stored.CalledAt != nil {
					t.Fatalf("failed call changed waiting state: %#v %v", stored, err)
				}
			})
		}
	}
}

func TestCallNextSelectsPublicCustomerFromNewNormalQueue(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	ctx := context.Background()
	joined, err := f.Service.RegisterCustomerQueue(ctx, queueapp.RegisterCustomerQueueInput{BusinessID: f.Business.ID, Username: "Guest", PhoneNumber: "081234567890"})
	if err != nil {
		t.Fatal(err)
	}
	f.Persist(*joined.Queue)
	counterID := uuid.New()
	r := &counterRepo{counter: counterdomain.Counter{ID: counterID, BusinessID: f.Business.ID}}
	s := counterapp.NewCounterService(r, f, nil, f.Redis)
	called, err := s.CallNextQueue(ctx, counterID, f.Business.ID)
	if err != nil || called == nil || called.ID != joined.Queue.ID || called.Priority || called.State != queuedomain.QueueStateCalled || called.CalledByCounterID == nil || *called.CalledByCounterID != counterID {
		t.Fatalf("public join/call-next contract: %#v %v", called, err)
	}
	if r.counter.CurrentQueueID == nil || *r.counter.CurrentQueueID != called.ID {
		t.Fatal("counter lost stable queue UUID")
	}
	if items, err := cache.GetWaitingQueues(ctx, f.Business.ID.String(), f.Redis); err != nil || len(items) != 0 {
		t.Fatalf("called customer remains waiting: %#v %v", items, err)
	}
	if !f.Server.Exists(cache.GuestPhoneKey(f.Business.ID.String(), joined.PhoneNumber)) {
		t.Fatal("called customer lost active reservation")
	}
}
