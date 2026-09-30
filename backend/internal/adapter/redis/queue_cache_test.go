package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"QueueLite/internal/queue/domain"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func waitingRedis(t *testing.T) *redis.Client {
	t.Helper()
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func seedWaiting(t *testing.T, rdb *redis.Client, key, prefix string, count int) {
	t.Helper()
	items := make([]redis.Z, count)
	for i := range items {
		items[i] = redis.Z{Member: fmt.Sprintf("%s-%03d", prefix, i), Score: float64(i)}
	}
	if err := rdb.ZAdd(context.Background(), key, items...).Err(); err != nil {
		t.Fatal(err)
	}
}

func TestFairWaitingSelection(t *testing.T) {
	for _, normalKey := range []string{NormalWaitingQueueKey("business"), LegacyWaitingQueueKey("business")} {
		t.Run(normalKey, func(t *testing.T) {
			rdb := waitingRedis(t)
			ctx := context.Background()
			priorityKey := PriorityWaitingQueueKey("business")
			seedWaiting(t, rdb, priorityKey, "priority", 12)
			seedWaiting(t, rdb, normalKey, "normal", 2)
			for i := 0; i < 12; i++ {
				item, err := PopNextFairWaitingQueueWithScore(ctx, rdb, "business")
				wantKey, wantStreak := priorityKey, i%6+1
				if i%6 == 5 {
					wantKey, wantStreak = normalKey, 0
				}
				if err != nil || item == nil || item.QueueKey != wantKey {
					t.Fatalf("selection %d: %#v %v, want %s", i, item, err, wantKey)
				}
				if got := rdb.Get(ctx, PriorityStreakKey("business")).Val(); got != fmt.Sprint(wantStreak) {
					t.Fatalf("selection %d streak = %s, want %d", i, got, wantStreak)
				}
			}
			// Priority-only traffic continues with the streak capped at five.
			for i := 0; i < 2; i++ {
				if item, err := PopNextFairWaitingQueueWithScore(ctx, rdb, "business"); err != nil || item == nil || item.QueueKey != priorityKey {
					t.Fatalf("priority-only selection: %#v %v", item, err)
				}
			}
			if got := rdb.Get(ctx, PriorityStreakKey("business")).Val(); got != "2" {
				t.Fatalf("priority-only streak = %s", got)
			}
			seedWaiting(t, rdb, priorityKey, "more-priority", 8)
			for i := 0; i < 8; i++ {
				if _, err := PopNextFairWaitingQueueWithScore(ctx, rdb, "business"); err != nil {
					t.Fatal(err)
				}
			}
			if got := rdb.Get(ctx, PriorityStreakKey("business")).Val(); got != "5" {
				t.Fatalf("uncapped priority-only streak = %s", got)
			}
			seedWaiting(t, rdb, priorityKey, "new-priority", 1)
			seedWaiting(t, rdb, normalKey, "new-normal", 1)
			item, err := PopNextFairWaitingQueueWithScore(ctx, rdb, "business")
			if err != nil || item == nil || item.QueueKey != normalKey {
				t.Fatalf("normal arriving at cap: %#v %v", item, err)
			}
			if _, err := PopNextFairWaitingQueueWithScore(ctx, rdb, "business"); err != nil {
				t.Fatal(err)
			}
			if item, err := PopNextFairWaitingQueueWithScore(ctx, rdb, "business"); err != nil || item != nil {
				t.Fatalf("empty queue: %#v %v", item, err)
			}
			// Streaks and waiting sets are isolated by business.
			seedWaiting(t, rdb, PriorityWaitingQueueKey("other"), "other", 1)
			item, err = PopNextFairWaitingQueueWithScore(ctx, rdb, "other")
			if err != nil || item == nil || rdb.Get(ctx, PriorityStreakKey("other")).Val() != "1" {
				t.Fatalf("business isolation: %#v %v", item, err)
			}
		})
	}
}

func TestConcurrentFairWaitingSelection(t *testing.T) {
	for _, normalKey := range []string{NormalWaitingQueueKey("business"), LegacyWaitingQueueKey("business")} {
		t.Run(normalKey, func(t *testing.T) {
			rdb := waitingRedis(t)
			ctx := context.Background()
			priorityKey := PriorityWaitingQueueKey("business")
			seedWaiting(t, rdb, priorityKey, "priority", 300)
			seedWaiting(t, rdb, normalKey, "normal", 60)
			// All 60 callers race on one business, as independent counters do.
			// Normal customers remain available throughout, so every six atomic
			// selections must contain exactly five priority and one normal.
			start := make(chan struct{})
			results := make(chan *WaitingQueueItem, 60)
			var wg sync.WaitGroup
			for i := 0; i < 60; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					item, err := PopNextFairWaitingQueueWithScore(ctx, rdb, "business")
					if err != nil || item == nil {
						t.Errorf("concurrent pop: %#v %v", item, err)
						return
					}
					results <- item
				}()
			}
			close(start)
			wg.Wait()
			close(results)
			seen := make(map[string]bool)
			priority, normal := 0, 0
			for item := range results {
				id := item.Member.(string)
				if seen[id] {
					t.Fatalf("duplicate selection %s", id)
				}
				seen[id] = true
				if item.QueueKey == priorityKey {
					priority++
				} else if item.QueueKey == normalKey {
					normal++
				} else {
					t.Fatalf("unexpected source %s", item.QueueKey)
				}
			}
			if priority != 50 || normal != 10 || rdb.Get(ctx, PriorityStreakKey("business")).Val() != "0" {
				t.Fatalf("fairness across counters: priority=%d normal=%d streak=%s", priority, normal, rdb.Get(ctx, PriorityStreakKey("business")).Val())
			}
		})
	}
}

func TestWaitingQueueRestoreAndRemoval(t *testing.T) {
	ctx := context.Background()
	rdb := waitingRedis(t)
	keys := []string{PriorityWaitingQueueKey("business"), NormalWaitingQueueKey("business"), LegacyWaitingQueueKey("business")}
	for _, key := range keys {
		seedWaiting(t, rdb, key, "ticket", 1)
		item, err := PopNextFairWaitingQueueWithScore(ctx, rdb, "business")
		if err != nil || item == nil || item.QueueKey != key || item.Score != 0 {
			t.Fatalf("pop source: %#v %v", item, err)
		}
		if err := RestoreWaitingQueueItem(ctx, rdb, *item); err != nil {
			t.Fatal(err)
		}
		for _, candidate := range keys {
			want := int64(0)
			if candidate == key {
				want = 1
			}
			if got := rdb.ZCard(ctx, candidate).Val(); got != want {
				t.Fatalf("restore to %s: got %d want %d", candidate, got, want)
			}
		}
		if err := RemoveWaitingQueue(ctx, rdb, "business", item.Member.(string)); err != nil {
			t.Fatal(err)
		}
	}
	// A migration duplicate must be removed from every applicable set.
	for _, key := range keys {
		seedWaiting(t, rdb, key, "duplicate", 1)
	}
	if err := RemoveWaitingQueue(ctx, rdb, "business", "duplicate-000"); err != nil {
		t.Fatal(err)
	}
	if items, err := GetWaitingQueues(ctx, "business", rdb); err != nil || len(items) != 0 {
		t.Fatalf("removal left waiting customers: %#v %v", items, err)
	}
}

func TestFairWaitingSelectionChecksTypesBeforePop(t *testing.T) {
	for _, key := range []string{PriorityStreakKey("business"), NormalWaitingQueueKey("business"), LegacyWaitingQueueKey("business")} {
		t.Run(key, func(t *testing.T) {
			rdb := waitingRedis(t)
			ctx := context.Background()
			seedWaiting(t, rdb, PriorityWaitingQueueKey("business"), "priority", 1)
			if err := rdb.Set(ctx, key, "invalid", 0).Err(); err != nil {
				t.Fatal(err)
			}
			if item, err := PopNextFairWaitingQueueWithScore(ctx, rdb, "business"); err == nil || item != nil {
				t.Fatalf("invalid cache accepted: %#v %v", item, err)
			}
			if got := rdb.ZCard(ctx, PriorityWaitingQueueKey("business")).Val(); got != 1 {
				t.Fatal("cache error lost a priority customer")
			}
		})
	}
}

func TestCustomerEnqueueUsesSplitWaitingKeys(t *testing.T) {
	for _, priority := range []bool{false, true} {
		t.Run(fmt.Sprint(priority), func(t *testing.T) {
			rdb := waitingRedis(t)
			ctx := context.Background()
			owner := uuid.New()
			q := domain.Queue{ID: uuid.New(), BusinessID: uuid.New(), UserID: &owner, Name: "A001", State: domain.QueueStateWaiting, Priority: priority, CreatedAt: time.Now().UTC()}
			customer := CustomerQueue{Queue: q, Username: "Customer", PhoneNumber: "6281234567890"}
			outcome, err := EnqueueCustomerQueue(ctx, rdb, customer)
			if err != nil || outcome != "OK" {
				t.Fatalf("enqueue: %s %v", outcome, err)
			}
			wantKey := NormalWaitingQueueKey(q.BusinessID.String())
			if priority {
				wantKey = PriorityWaitingQueueKey(q.BusinessID.String())
			}
			item, err := PopNextFairWaitingQueueWithScore(ctx, rdb, q.BusinessID.String())
			if err != nil || item == nil || item.QueueKey != wantKey || item.Member != q.ID.String() || item.Score != QueueScore(q) {
				t.Fatalf("registration/call-next mismatch: %#v %v", item, err)
			}
			if got := rdb.ZCard(ctx, LegacyWaitingQueueKey(q.BusinessID.String())).Val(); got != 0 {
				t.Fatal("registration wrote obsolete waiting key")
			}
		})
	}
}
