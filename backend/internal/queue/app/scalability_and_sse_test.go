package app_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/config"
	queueapp "QueueLite/internal/queue/app"
	"QueueLite/internal/queue/domain"
	"QueueLite/internal/testutil"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type redisCapacityQuotaFixture struct {
	*testutil.QueueFixture
	remaining   int
	seedReads   atomic.Int32
	floorCalls  atomic.Int32
	syncRecords sync.Map
}

func (f *redisCapacityQuotaFixture) GetRemainingBusinessCapacity(_ context.Context, _ uuid.UUID) (int, error) {
	f.seedReads.Add(1)
	return f.remaining, nil
}

func (f *redisCapacityQuotaFixture) GetDailyQueueNumberFloor(_ context.Context, _ uuid.UUID, _ time.Time) (int64, error) {
	f.floorCalls.Add(1)
	return 0, nil
}

func TestRedisAtomicCapacityEnforcementAndCacheMissFloor(t *testing.T) {
	base := testutil.NewQueueFixture(t)
	wrapped := &redisCapacityQuotaFixture{QueueFixture: base, remaining: 5}
	svc := queueapp.NewQueueService(wrapped, wrapped, base.Redis, base, base, base)

	ctx := context.Background()
	const attempts = 12
	start := make(chan struct{})
	var succeeded atomic.Int32
	var full atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		phone := fmt.Sprintf("08123456%04d", i)
		go func(idx int, p string) {
			defer wg.Done()
			<-start
			_, err := svc.RegisterCustomerQueue(ctx, queueapp.RegisterCustomerQueueInput{
				BusinessID:  base.Business.ID,
				Username:    fmt.Sprintf("Guest-%d", idx),
				PhoneNumber: p,
			})
			if err == nil {
				succeeded.Add(1)
				return
			}
			full.Add(1)
		}(i, phone)
	}
	close(start)
	wg.Wait()

	if succeeded.Load() != 5 {
		t.Fatalf("expected exactly 5 admitted queues, got %d (rejected=%d)", succeeded.Load(), full.Load())
	}
	if full.Load() != attempts-5 {
		t.Fatalf("expected %d full rejections, got %d", attempts-5, full.Load())
	}

	// Sequential warm joins should not query GetDailyQueueNumberFloor once Redis counter exists.
	beforeFloor := wrapped.floorCalls.Load()
	if err := cache.SeedBusinessCapacity(ctx, base.Redis, base.Business.ID.String(), time.Now().UTC(), 10); err != nil {
		t.Fatal(err)
	}
	// Reset capacity key to 10 to test a warm join without hitting floor query.
	base.Redis.Set(ctx, cache.DailyBusinessCapacityKey(base.Business.ID.String(), time.Now().UTC()), 10, time.Hour)
	if _, err := svc.RegisterCustomerQueue(ctx, queueapp.RegisterCustomerQueueInput{
		BusinessID:  base.Business.ID,
		Username:    "WarmGuest",
		PhoneNumber: "081299998888",
	}); err != nil {
		t.Fatalf("warm join failed: %v", err)
	}
	if wrapped.floorCalls.Load() != beforeFloor {
		t.Fatalf("warm join unexpectedly queried DB floor: before=%d after=%d", beforeFloor, wrapped.floorCalls.Load())
	}
}

type poisonPillRepo struct {
	queueapp.QueueRepo
	poisonID  uuid.UUID
	mu        sync.Mutex
	persisted map[uuid.UUID]domain.Queue
}

func (r *poisonPillRepo) CreateQueues(_ context.Context, queues []domain.Queue) error {
	for _, q := range queues {
		if q.ID == r.poisonID {
			return errors.New("foreign key violation")
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, q := range queues {
		r.persisted[q.ID] = q
	}
	return nil
}

type recordingCapacitySyncer struct {
	mu     sync.Mutex
	counts map[uuid.UUID]int
}

func (s *recordingCapacitySyncer) ConsumeBusinessCapacityBatch(_ context.Context, businessID uuid.UUID, amount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[businessID] += amount
	return nil
}

func TestWorkerPoolIsolatesPoisonPillAndSyncsCapacityBatch(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cache.EnsureQueueConsumerGroup(ctx, rdb); err != nil {
		t.Fatal(err)
	}

	bizID := uuid.New()
	good1 := domain.Queue{ID: uuid.New(), BusinessID: bizID, Name: "A001", State: domain.QueueStateWaiting, CreatedAt: time.Now().UTC()}
	poison := domain.Queue{ID: uuid.New(), BusinessID: bizID, Name: "A002", State: domain.QueueStateWaiting, CreatedAt: time.Now().UTC()}
	good2 := domain.Queue{ID: uuid.New(), BusinessID: bizID, Name: "A003", State: domain.QueueStateWaiting, CreatedAt: time.Now().UTC()}

	for _, q := range []domain.Queue{good1, poison, good2} {
		if err := cache.AddQueueStream(ctx, rdb, q); err != nil {
			t.Fatal(err)
		}
	}

	repo := &poisonPillRepo{poisonID: poison.ID, persisted: make(map[uuid.UUID]domain.Queue)}
	syncer := &recordingCapacitySyncer{counts: make(map[uuid.UUID]int)}

	done := make(chan struct{})
	go func() {
		defer close(done)
		queueapp.RunDatabaseWorkerPool(ctx, rdb, repo, queueapp.DatabaseWorkerOptions{
			WorkerCount:     1,
			BatchSize:       10,
			ReclaimIdle:     0,
			ReclaimInterval: 5 * time.Millisecond,
			RetryDelay:      5 * time.Millisecond,
			MaxDeliveries:   2,
			CapacitySyncer:  syncer,
		})
	}()

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		dlqLen := rdb.XLen(ctx, config.QueueDeadLetterStreamName).Val()
		pending := rdb.XPending(ctx, config.StreamName, config.QueueConsumerGroup).Val()
		repo.mu.Lock()
		persistedCount := len(repo.persisted)
		repo.mu.Unlock()
		if persistedCount == 2 && dlqLen == 1 && pending != nil && pending.Count == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	rdb.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker pool did not shut down cleanly")
	}

	repo.mu.Lock()
	_, hasGood1 := repo.persisted[good1.ID]
	_, hasGood2 := repo.persisted[good2.ID]
	_, hasPoison := repo.persisted[poison.ID]
	repo.mu.Unlock()
	if !hasGood1 || !hasGood2 || hasPoison {
		t.Fatalf("unexpected persistence isolation: good1=%v good2=%v poison=%v", hasGood1, hasGood2, hasPoison)
	}

	syncer.mu.Lock()
	synced := syncer.counts[bizID]
	syncer.mu.Unlock()
	if synced != 2 {
		t.Fatalf("expected capacity sync count 2, got %d", synced)
	}
}

func TestGetWaitingQueueSnapshotUsesMGetBatch(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	q1 := domain.Queue{ID: uuid.New(), BusinessID: f.Business.ID, Name: "A001", State: domain.QueueStateWaiting, CreatedAt: now}
	q2 := domain.Queue{ID: uuid.New(), BusinessID: f.Business.ID, Name: "A002", State: domain.QueueStateWaiting, CreatedAt: now.Add(time.Second)}
	if err := cache.AddWaitingQueue(ctx, f.Redis, q1); err != nil {
		t.Fatal(err)
	}
	if err := cache.AddWaitingQueue(ctx, f.Redis, q2); err != nil {
		t.Fatal(err)
	}

	snapshot, err := f.Service.GetWaitingQueueSnapshot(ctx, f.Business.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot) != 2 || snapshot[0].QueueName != "A001" || snapshot[1].QueueName != "A002" {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}
