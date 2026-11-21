package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	cache "QueueLite/internal/adapter/redis"
	telebotapp "QueueLite/internal/bot/app"
	counterapp "QueueLite/internal/counter/app"
	counterdomain "QueueLite/internal/counter/domain"
	queuedomain "QueueLite/internal/queue/domain"
	"QueueLite/internal/testutil"

	"github.com/google/uuid"
)

type recordingNotifier struct {
	events []struct {
		userID uuid.UUID
		text   string
	}
}

func (n *recordingNotifier) EnqueueUserNotification(_ context.Context, userID uuid.UUID, text string) error {
	n.events = append(n.events, struct {
		userID uuid.UUID
		text   string
	}{userID: userID, text: text})
	return nil
}

func TestCallNextNotifiesCalledAndNextUsersAndHandlesLastCustomer(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()
	now := time.Now().UTC()
	q1 := queuedomain.Queue{ID: uuid.New(), BusinessID: f.Business.ID, UserID: &user1, Name: "A001", State: queuedomain.QueueStateWaiting, CreatedAt: now}
	q2 := queuedomain.Queue{ID: uuid.New(), BusinessID: f.Business.ID, UserID: &user2, Name: "A002", State: queuedomain.QueueStateWaiting, CreatedAt: now.Add(time.Second)}
	f.Persist(q1)
	f.Persist(q2)
	if err := cache.AddWaitingQueue(ctx, f.Redis, q1); err != nil {
		t.Fatal(err)
	}
	if err := cache.AddWaitingQueue(ctx, f.Redis, q2); err != nil {
		t.Fatal(err)
	}

	counterID := uuid.New()
	repo := &counterRepo{counter: counterdomain.Counter{ID: counterID, BusinessID: f.Business.ID}}
	notifier := &recordingNotifier{}
	svc := counterapp.NewCounterService(repo, f, nil, f.Redis)
	svc.SetNotifier(notifier)

	called1, err := svc.CallNextQueue(ctx, counterID, f.Business.ID)
	if err != nil || called1 == nil || called1.ID != q1.ID {
		t.Fatalf("first call failed: %+v %v", called1, err)
	}
	if len(notifier.events) != 2 {
		t.Fatalf("expected 2 notifications (called + next), got %+v", notifier.events)
	}
	if notifier.events[0].userID != user1 || notifier.events[0].text != telebotapp.CurrentQueueNotificationMessage {
		t.Fatalf("called notification sent to wrong user: %+v", notifier.events[0])
	}
	if notifier.events[1].userID != user2 || notifier.events[1].text != telebotapp.NextQueueNotificationMessage {
		t.Fatalf("next notification sent to wrong user: %+v", notifier.events[1])
	}

	// Complete first and call the last remaining customer (must not panic on empty waiting queue).
	if err := svc.RemoveQueueFromCounter(ctx, counterID, q1.ID); err != nil {
		t.Fatal(err)
	}
	notifier.events = nil
	called2, err := svc.CallNextQueue(ctx, counterID, f.Business.ID)
	if err != nil || called2 == nil || called2.ID != q2.ID {
		t.Fatalf("second call failed: %+v %v", called2, err)
	}
	if len(notifier.events) != 1 || notifier.events[0].userID != user2 || notifier.events[0].text != telebotapp.CurrentQueueNotificationMessage {
		t.Fatalf("last customer notification mismatch: %+v", notifier.events)
	}
}

func TestSkipQueueDoesNotOverwriteSkippedToDoneWhenCounterClearFails(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	ctx := context.Background()
	counterID := uuid.New()
	q := queuedomain.Queue{
		ID:                uuid.New(),
		BusinessID:        f.Business.ID,
		Name:              "A001",
		State:             queuedomain.QueueStateCalled,
		CalledByCounterID: &counterID,
		CreatedAt:         time.Now().UTC(),
	}
	f.Persist(q)
	repo := &failingAssignmentRepo{
		counterRepo:     &counterRepo{counter: counterdomain.Counter{ID: counterID, BusinessID: f.Business.ID, CurrentQueueID: &q.ID}},
		assignmentError: errors.New("counter update failed"),
	}
	svc := counterapp.NewCounterService(repo, f, nil, f.Redis)
	if _, err := svc.SkipQueue(ctx, counterID, q.ID); err == nil {
		t.Fatal("expected skip to propagate counter clear error")
	}
	stored, _ := f.GetQueue(ctx, q.ID)
	if stored.State == queuedomain.QueueStateCompleted {
		t.Fatalf("skip overwrote queue state to done: %+v", stored)
	}
}
