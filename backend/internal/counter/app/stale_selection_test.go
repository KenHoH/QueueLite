package app_test

import (
	cache "QueueLite/internal/adapter/redis"
	counterapp "QueueLite/internal/counter/app"
	counterdomain "QueueLite/internal/counter/domain"
	queuedomain "QueueLite/internal/queue/domain"
	"QueueLite/internal/testutil"
	"context"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"sync"
	"testing"
	"time"
)

func TestCallNextRejectsStaleAndCrossBusinessWaitingEntries(t *testing.T) {
	for _, state := range []queuedomain.QueueState{queuedomain.QueueStateCancelled, queuedomain.QueueStateCompleted, queuedomain.QueueStateProcessing, queuedomain.QueueStateCalled, queuedomain.QueueStateSkipped, "cross-business"} {
		t.Run(string(state), func(t *testing.T) {
			f := testutil.NewQueueFixture(t)
			q := queuedomain.Queue{ID: uuid.New(), BusinessID: f.Business.ID, Name: "A001", State: state, CreatedAt: time.Now()}
			if state == "cross-business" {
				q.State = queuedomain.QueueStateWaiting
				q.BusinessID = uuid.New()
			}
			f.Persist(q)
			key := cache.NormalWaitingQueueKey(f.Business.ID.String())
			f.Redis.ZAdd(context.Background(), key, redis.Z{Member: q.ID.String(), Score: 1})
			repo := &counterRepo{counter: counterdomain.Counter{ID: uuid.New(), BusinessID: f.Business.ID}}
			result, err := counterapp.NewCounterService(repo, f, nil, f.Redis).CallNextQueue(context.Background(), repo.counter.ID, f.Business.ID)
			if err == nil || result != nil || repo.counter.CurrentQueueID != nil {
				t.Fatal("stale queue assigned")
			}
			stored, _ := f.GetQueue(context.Background(), q.ID)
			if stored.State != q.State || stored.CalledByCounterID != nil {
				t.Fatal("stale queue resurrected")
			}
			if f.Redis.ZCard(context.Background(), key).Val() != 0 {
				t.Fatal("invalid waiting item restored")
			}
		})
	}
}

func TestConcurrentCountersCannotConsumeMigrationDuplicateTwice(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	ctx := context.Background()
	q := queuedomain.Queue{ID: uuid.New(), BusinessID: f.Business.ID, Name: "A001", State: queuedomain.QueueStateWaiting, CreatedAt: time.Now()}
	f.Persist(q)
	for _, key := range []string{cache.NormalWaitingQueueKey(f.Business.ID.String()), cache.LegacyWaitingQueueKey(f.Business.ID.String())} {
		f.Redis.ZAdd(ctx, key, redis.Z{Member: q.ID.String(), Score: 1})
	}
	start := make(chan struct{})
	results := make(chan bool, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			repo := &counterRepo{counter: counterdomain.Counter{ID: uuid.New(), BusinessID: f.Business.ID}}
			s := counterapp.NewCounterService(repo, f, nil, f.Redis)
			<-start
			result, err := s.CallNextQueue(ctx, repo.counter.ID, f.Business.ID)
			results <- err == nil && result != nil
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	successes := 0
	for success := range results {
		if success {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("queue consumed %d times", successes)
	}
}

func TestCancellationAndCallNextHaveOneWaitingStateWinner(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	ctx := context.Background()
	for i := 0; i < 40; i++ {
		q := queuedomain.Queue{ID: uuid.New(), BusinessID: f.Business.ID, Name: "A001", State: queuedomain.QueueStateWaiting, CreatedAt: time.Now()}
		f.Persist(q)
		cache.AddWaitingQueue(ctx, f.Redis, q)
		repo := &counterRepo{counter: counterdomain.Counter{ID: uuid.New(), BusinessID: f.Business.ID}}
		s := counterapp.NewCounterService(repo, f, nil, f.Redis)
		start := make(chan struct{})
		results := make(chan bool, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); <-start; results <- f.Service.CancelWaitingQueue(ctx, q.ID) == nil }()
		go func() {
			defer wg.Done()
			<-start
			called, err := s.CallNextQueue(ctx, repo.counter.ID, f.Business.ID)
			results <- err == nil && called != nil
		}()
		close(start)
		wg.Wait()
		close(results)
		wins := 0
		for won := range results {
			if won {
				wins++
			}
		}
		if wins != 1 {
			t.Fatalf("race had %d winners", wins)
		}
		stored, _ := f.GetQueue(ctx, q.ID)
		if stored.State == queuedomain.QueueStateCancelled && repo.counter.CurrentQueueID != nil {
			t.Fatal("cancelled queue assigned")
		}
		if stored.State == queuedomain.QueueStateCalled && repo.counter.CurrentQueueID == nil {
			t.Fatal("called queue lost counter")
		}
	}
}
