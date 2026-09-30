package cache

import (
	"QueueLite/internal/config"
	"context"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"sync"
	"testing"
	"time"
)

func TestDailyQueueNumberAtomic(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()
	ctx := context.Background()
	day := time.Date(2026, 9, 30, 23, 59, 0, 0, time.UTC)
	for i := 1; i <= 3; i++ {
		name, err := GenerateDailyQueueName(ctx, rdb, "business", day, 0)
		if err != nil || name != fmt.Sprintf("A%03d", i) {
			t.Fatalf("sequence: %s %v", name, err)
		}
	}
	var wg sync.WaitGroup
	names := make(chan string, 64)
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name, err := GenerateDailyQueueName(ctx, rdb, "business", day, 0)
			if err != nil {
				t.Error(err)
				return
			}
			names <- name
		}()
	}
	wg.Wait()
	close(names)
	seen := make(map[string]bool)
	for name := range names {
		if seen[name] {
			t.Fatalf("duplicate %s", name)
		}
		seen[name] = true
	}
	if len(seen) != 64 {
		t.Fatalf("got %d allocations", len(seen))
	}
	if !seen["A004"] || !seen["A067"] {
		t.Fatal("concurrent sequence has gaps")
	}
	for _, tc := range []struct {
		business string
		day      time.Time
		floor    int64
		want     string
	}{
		{"other", day, 0, "A001"}, {"business", day.Add(time.Minute), 0, "A001"},
		{"business", day, 100, "A101"}, {"business", day, 0, "A102"},
	} {
		got, err := GenerateDailyQueueName(ctx, rdb, tc.business, tc.day, tc.floor)
		if err != nil || got != tc.want {
			t.Fatalf("scoping/floor: %s %v", got, err)
		}
	}
	if ttl := server.TTL(DailyQueueCounterKey("business", day)); ttl != 48*time.Hour {
		t.Fatalf("ttl: %s", ttl)
	}
	// Counter loss must not reuse a label still waiting in the async stream.
	rdb.XAdd(ctx, &redis.XAddArgs{Stream: config.StreamName, Values: map[string]any{"business_id": "business", "created_at": day.Format(time.RFC3339Nano), "name": "A250"}})
	rdb.Del(ctx, DailyQueueCounterKey("business", day))
	got, err := GenerateDailyQueueName(ctx, rdb, "business", day, 100)
	if err != nil || got != "A251" {
		t.Fatalf("stream recovery: %s %v", got, err)
	}
}
