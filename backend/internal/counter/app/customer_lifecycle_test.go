package app_test

import (
	cache "QueueLite/internal/adapter/redis"
	counterapp "QueueLite/internal/counter/app"
	counterdomain "QueueLite/internal/counter/domain"
	queueapp "QueueLite/internal/queue/app"
	queuedomain "QueueLite/internal/queue/domain"
	"QueueLite/internal/testutil"
	"context"
	"github.com/google/uuid"
	"testing"
)

type counterRepo struct {
	counterapp.CounterRepo
	counter counterdomain.Counter
}

func (r *counterRepo) GetCounter(context.Context, uuid.UUID) (*counterdomain.Counter, error) {
	return &r.counter, nil
}
func (r *counterRepo) UpdateCounterCustomer(_ context.Context, _ uuid.UUID, q *uuid.UUID) error {
	r.counter.CurrentQueueID = q
	return nil
}

func TestCounterTerminalTransitionsReleasePhone(t *testing.T) {
	for _, action := range []string{"done", "skip", "call-next"} {
		t.Run(action, func(t *testing.T) {
			f := testutil.NewQueueFixture(t)
			ctx := context.Background()
			result, err := f.Service.RegisterCustomerQueue(ctx, queueapp.RegisterCustomerQueueInput{BusinessID: f.Business.ID, Username: "Guest", PhoneNumber: "081234567890"})
			if err != nil {
				t.Fatal(err)
			}
			counterID := uuid.New()
			q := *result.Queue
			q.State = queuedomain.QueueStateCalled
			q.CalledByCounterID = &counterID
			f.Persist(q)
			if err := cache.RemoveWaitingQueue(ctx, f.Redis, q.BusinessID.String(), q.ID.String()); err != nil {
				t.Fatal(err)
			}
			// Stale migration copies must disappear on every terminal path.
			for _, key := range []string{cache.PriorityWaitingQueueKey(q.BusinessID.String()), cache.NormalWaitingQueueKey(q.BusinessID.String()), cache.LegacyWaitingQueueKey(q.BusinessID.String())} {
				if err := cache.AddQueue(ctx, f.Redis, key, q); err != nil {
					t.Fatal(err)
				}
			}
			r := &counterRepo{counter: counterdomain.Counter{ID: counterID, BusinessID: q.BusinessID, CurrentQueueID: &q.ID}}
			s := counterapp.NewCounterService(r, f, nil, f.Redis)
			switch action {
			case "done":
				err = s.RemoveQueueFromCounter(ctx, counterID, q.ID)
			case "skip":
				_, err = s.SkipQueue(ctx, counterID, q.ID)
			case "call-next":
				_, err = s.CallNextQueue(ctx, counterID, q.BusinessID)
			}
			if err != nil {
				t.Fatal(err)
			}
			stored, _ := f.GetQueue(ctx, q.ID)
			if action == "skip" && stored.CancelledAt != nil {
				t.Fatal("skip fabricated cancellation timestamp")
			}
			if action != "skip" && stored.DoneAt == nil {
				t.Fatal("counter completion missing done timestamp")
			}
			if f.Server.Exists(cache.GuestPhoneKey(q.BusinessID.String(), result.PhoneNumber)) {
				t.Fatal("counter terminal queue retained phone")
			}
			waiting, err := cache.GetWaitingQueues(ctx, q.BusinessID.String(), f.Redis)
			if err != nil || len(waiting) != 0 {
				t.Fatalf("terminal queue remains waiting: %#v %v", waiting, err)
			}
		})
	}
}

func TestCounterCalledProcessingDoneTimestamps(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	ctx := context.Background()
	result, err := f.Service.RegisterCustomerQueue(ctx, queueapp.RegisterCustomerQueueInput{BusinessID: f.Business.ID, Username: "Guest", PhoneNumber: "081234567890"})
	if err != nil {
		t.Fatal(err)
	}
	f.Persist(*result.Queue)
	repo := &counterRepo{counter: counterdomain.Counter{ID: uuid.New(), BusinessID: f.Business.ID}}
	service := counterapp.NewCounterService(repo, f, nil, f.Redis)
	called, err := service.CallNextQueue(ctx, repo.counter.ID, f.Business.ID)
	if err != nil || called == nil || called.CalledAt == nil {
		t.Fatalf("call missing timestamp: %+v %v", called, err)
	}
	processed, err := service.ProcessCalledQueue(ctx, repo.counter.ID, called.ID)
	if err != nil || processed.ProcessingAt == nil || processed.CalledAt == nil {
		t.Fatalf("processing missing timestamps: %+v %v", processed, err)
	}
	if err := service.RemoveQueueFromCounter(ctx, repo.counter.ID, called.ID); err != nil {
		t.Fatal(err)
	}
	done, _ := f.GetQueue(ctx, called.ID)
	if done.DoneAt == nil || done.DoneAt.Before(*done.ProcessingAt) || done.ProcessingAt.Before(*done.CalledAt) || done.CancelledAt != nil {
		t.Fatalf("invalid counter timestamps: %+v", done)
	}
}
