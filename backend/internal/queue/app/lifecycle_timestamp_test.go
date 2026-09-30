package app_test

import (
	"QueueLite/internal/queue/domain"
	"QueueLite/internal/testutil"
	"context"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestLifecycleMethodsStampCorrectEventAndPreserveRetries(t *testing.T) {
	for _, state := range []domain.QueueState{domain.QueueStateCalled, domain.QueueStateProcessing, domain.QueueStateCompleted, domain.QueueStateCancelled, domain.QueueStateSkipped} {
		t.Run(string(state), func(t *testing.T) {
			f := testutil.NewQueueFixture(t)
			q := domain.Queue{ID: uuid.New(), BusinessID: f.Business.ID, Name: "A001", State: domain.QueueStateWaiting}
			f.Persist(q)
			ctx := context.Background()
			before := time.Now()
			var err error
			switch state {
			case domain.QueueStateProcessing:
				err = f.Service.MarkAsProcessing(ctx, q.ID)
			case domain.QueueStateCompleted:
				err = f.Service.MarkAsDone(ctx, q.ID)
			case domain.QueueStateCancelled:
				err = f.Service.MarkAsCancelled(ctx, q.ID)
			case domain.QueueStateSkipped:
				err = f.Service.MarkAsSkipped(ctx, q.ID)
			default:
				err = f.Service.UpdateState(ctx, q.ID, state)
			}
			if err != nil {
				t.Fatal(err)
			}
			stored, _ := f.GetQueue(ctx, q.ID)
			stamps := map[domain.QueueState]*time.Time{domain.QueueStateCalled: stored.CalledAt, domain.QueueStateProcessing: stored.ProcessingAt, domain.QueueStateCompleted: stored.DoneAt, domain.QueueStateCancelled: stored.CancelledAt}
			for event, stamp := range stamps {
				if event == state {
					if stamp == nil || stamp.Before(before) || stamp.After(time.Now()) {
						t.Fatalf("event not stamped: %s %+v", state, stored)
					}
				} else if stamp != nil {
					t.Fatalf("fabricated event %s for %s", event, state)
				}
			}
			if err := f.Service.UpdateState(ctx, q.ID, state); err != nil {
				t.Fatal(err)
			}
			again, _ := f.GetQueue(ctx, q.ID)
			if state == domain.QueueStateCancelled && !again.CancelledAt.Equal(*stored.CancelledAt) {
				t.Fatal("retry overwrote event")
			}
		})
	}
}
