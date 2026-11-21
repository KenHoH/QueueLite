package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const DefaultCounterLockTTL = 30 * time.Second

var releaseCounterLockScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0
`)

func CounterOperationLockKey(counterID string) string {
	return "counter:operation:" + counterID
}

// AcquireCounterLock acquires a compare-and-delete Redis lease for a single counter.
// Returns (releaseFn, true, nil) when acquired, or (nil, false, nil) if already locked.
func AcquireCounterLock(ctx context.Context, rdb *redis.Client, counterID string, ttl time.Duration) (func(), bool, error) {
	if rdb == nil {
		return func() {}, true, nil
	}
	if ttl <= 0 {
		ttl = DefaultCounterLockTTL
	}
	key := CounterOperationLockKey(counterID)
	token := uuid.NewString()
	locked, err := rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, false, err
	}
	if !locked {
		return nil, false, nil
	}
	release := func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = releaseCounterLockScript.Run(releaseCtx, rdb, []string{key}, token).Err()
	}
	return release, true, nil
}
