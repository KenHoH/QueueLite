package app

import (
	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/config"
	"QueueLite/internal/queue/domain"
	"context"
	"errors"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"sync"
	"testing"
	"time"
)

type workerRepo struct {
	QueueRepo
	mu        sync.Mutex
	calls     int
	failFirst bool
	persisted chan []domain.Queue
}

func (r *workerRepo) CreateQueues(ctx context.Context, queues []domain.Queue) error {
	r.mu.Lock()
	r.calls++
	fail := r.failFirst && r.calls == 1
	r.mu.Unlock()
	if fail {
		return errors.New("temporary database failure")
	}
	select {
	case r.persisted <- queues:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func TestWorkerRecoversFailedAndInterruptedBatches(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		t.Run(map[bool]string{false: "write-failure", true: "interrupted-consumer"}[interrupted], func(t *testing.T) {
			server := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: server.Addr()})
			defer client.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			q := domain.Queue{ID: uuid.New(), BusinessID: uuid.New(), Name: "A001", State: domain.QueueStateWaiting, Priority: true, CreatedAt: time.Now().UTC()}
			if err := cache.EnsureQueueConsumerGroup(ctx, client); err != nil {
				t.Fatal(err)
			}
			if err := cache.AddQueueStream(ctx, client, q); err != nil {
				t.Fatal(err)
			}
			if interrupted {
				if err := client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: config.QueueConsumerGroup, Consumer: "interrupted", Streams: []string{config.StreamName, ">"}, Count: 10}).Err(); err != nil {
					t.Fatal(err)
				}
			}
			repo := &workerRepo{failFirst: !interrupted, persisted: make(chan []domain.Queue, 4)}
			done := make(chan struct{})
			go func() { defer close(done); runDatabaseWorkerStream(ctx, client, repo, 0, time.Millisecond) }()
			select {
			case batch := <-repo.persisted:
				if len(batch) != 1 || batch[0].ID != q.ID || !batch[0].Priority {
					t.Fatalf("lost queue fields: %#v", batch)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("pending message never recovered")
			}
			until := time.Now().Add(time.Second)
			for client.XPending(ctx, config.StreamName, config.QueueConsumerGroup).Val().Count != 0 && time.Now().Before(until) {
				time.Sleep(time.Millisecond)
			}
			if pending := client.XPending(ctx, config.StreamName, config.QueueConsumerGroup).Val(); pending.Count != 0 {
				t.Fatal("persisted batch was not acknowledged")
			}
			if client.XLen(ctx, config.StreamName).Val() != 1 {
				t.Fatal("source stream unexpectedly trimmed")
			}
			cancel()
			client.Close()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("worker did not stop on cancellation")
			}
		})
	}
}

func TestWorkerDoesNotAckFailedDeadLetter(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := cache.EnsureQueueConsumerGroup(ctx, client); err != nil {
		t.Fatal(err)
	}
	client.Set(ctx, config.QueueDeadLetterStreamName, "wrong-type", 0)
	client.XAdd(ctx, &redis.XAddArgs{Stream: config.StreamName, Values: map[string]any{"id": "invalid"}})
	done := make(chan struct{})
	go func() {
		defer close(done)
		runDatabaseWorkerStream(ctx, client, &workerRepo{}, time.Hour, time.Millisecond)
	}()
	until := time.Now().Add(3 * time.Second)
	for client.XPending(ctx, config.StreamName, config.QueueConsumerGroup).Val().Count == 0 && time.Now().Before(until) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond)
	if client.XPending(ctx, config.StreamName, config.QueueConsumerGroup).Val().Count != 1 {
		t.Fatal("invalid message lost before dead letter recorded")
	}
	cancel()
	client.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}
