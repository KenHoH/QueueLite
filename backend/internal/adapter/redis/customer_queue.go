package cache

import (
	"QueueLite/internal/config"
	"QueueLite/internal/queue/domain"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func DailyQueueCounterKey(businessID string, day time.Time) string {
	return "queue:daily:" + businessID + ":" + day.UTC().Format("2006-01-02")
}

func DailyBusinessCapacityKey(businessID string, day time.Time) string {
	return "queue:capacity:" + businessID + ":" + day.UTC().Format("2006-01-02")
}

func HasDailyQueueCounter(ctx context.Context, rdb *redis.Client, businessID string, day time.Time) (bool, error) {
	if rdb == nil {
		return false, nil
	}
	count, err := rdb.Exists(ctx, DailyQueueCounterKey(businessID, day)).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SeedBusinessCapacity initializes today's Redis capacity key only if absent so concurrent
// joins cannot reset an already-decremented counter.
func SeedBusinessCapacity(ctx context.Context, rdb *redis.Client, businessID string, day time.Time, capacity int) error {
	if rdb == nil {
		return nil
	}
	return rdb.SetNX(ctx, DailyBusinessCapacityKey(businessID, day), capacity, 48*time.Hour).Err()
}

// InvalidateBusinessCapacity clears today's cached capacity key (e.g. after plan changes).
func InvalidateBusinessCapacity(ctx context.Context, rdb *redis.Client, businessID string, day time.Time) error {
	if rdb == nil {
		return nil
	}
	return rdb.Del(ctx, DailyBusinessCapacityKey(businessID, day)).Err()
}

// The DB maximum is a floor, never a replacement for Redis's high-water mark.
// Atomic initialization/increment avoids the async DB-count race.
// On cache miss, scan only a bounded recent stream tail (500 entries) so Lua stays O(1).
var dailyQueueNumber = redis.NewScript(`
local stored = redis.call('GET', KEYS[1])
local current = tonumber(stored or '0')
local floor = tonumber(ARGV[1])
if not stored then
  for _,entry in ipairs(redis.call('XREVRANGE', KEYS[2], '+', '-', 'COUNT', 500)) do
    local fields = entry[2]
    local business, day, name
    for i=1,#fields,2 do
      if fields[i] == 'business_id' then business = fields[i+1] end
      if fields[i] == 'created_at' then day = string.sub(fields[i+1], 1, 10) end
      if fields[i] == 'name' then name = fields[i+1] end
    end
    if business == ARGV[2] and day == ARGV[3] and name then
      local n = tonumber(string.match(name, '^A(%d+)$'))
      if n and n > floor then floor = n end
    end
  end
end
if floor > current then redis.call('SET', KEYS[1], floor) end
local next = redis.call('INCR', KEYS[1])
redis.call('EXPIRE', KEYS[1], 172800)
return next
`)

func GenerateDailyQueueName(ctx context.Context, rdb *redis.Client, businessID string, day time.Time, floor int64) (string, error) {
	n, err := dailyQueueNumber.Run(ctx, rdb, []string{DailyQueueCounterKey(businessID, day), config.StreamName}, floor, businessID, day.UTC().Format("2006-01-02")).Int64()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("A%03d", n), nil
}

type CustomerQueue struct {
	Queue       domain.Queue `json:"queue"`
	Username    string       `json:"username"`
	PhoneNumber string       `json:"phoneNumber"`
	Guest       bool         `json:"guest"`
}

func CustomerQueueKey(queueID string) string { return "queue:customer:" + queueID }
func ActiveQueueOwnerKey(businessID, ownerID string) string {
	return "queue:active_owner:" + businessID + ":" + ownerID
}

// All registration-side Redis state, optional atomic daily capacity decrement, and the
// bounded stream entry are accepted together in a single Lua execution.
// Key type checks precede writes because Lua runtime errors do not roll back writes.
var enqueueCustomer = redis.NewScript(`
local expected = {'string','string','string','string','zset','stream'}
for i=1,6 do
  local kind = redis.call('TYPE', KEYS[i]).ok
  if kind ~= 'none' and kind ~= expected[i] then return 'CACHE_KEY_TYPE_ERROR' end
end
local checkCapacity = ARGV[11] == '1'
if checkCapacity then
  local capKind = redis.call('TYPE', KEYS[7]).ok
  if capKind ~= 'none' and capKind ~= 'string' then return 'CACHE_KEY_TYPE_ERROR' end
end
if redis.call('EXISTS', KEYS[3]) == 1 then return 'OK' end
if redis.call('EXISTS', KEYS[2]) == 1 then return 'ACTIVE_QUEUE_EXISTS' end
if redis.call('EXISTS', KEYS[1]) == 1 then return 'PHONE_ALREADY_REGISTERED' end
if checkCapacity then
  local remaining = redis.call('GET', KEYS[7])
  if not remaining then return 'CAPACITY_NOT_SEEDED' end
  local cap = tonumber(remaining)
  if not cap or cap <= 0 then return 'BUSINESS_QUEUE_FULL' end
  redis.call('DECR', KEYS[7])
  redis.call('EXPIRE', KEYS[7], 172800)
end
redis.call('XADD', KEYS[6], 'MAXLEN', '~', ARGV[12], '*', 'id', ARGV[1], 'business_id', ARGV[2], 'user_id', ARGV[3], 'name', ARGV[4], 'priority', ARGV[5], 'state', ARGV[6], 'created_at', ARGV[7])
redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[10])
redis.call('SET', KEYS[2], ARGV[1], 'EX', ARGV[10])
redis.call('SET', KEYS[3], ARGV[8], 'EX', ARGV[10])
redis.call('SET', KEYS[4], ARGV[4], 'EX', ARGV[10])
redis.call('ZADD', KEYS[5], ARGV[9], ARGV[1])
return 'OK'
`)

func EnqueueCustomerQueue(ctx context.Context, rdb *redis.Client, customer CustomerQueue) (string, error) {
	return enqueueCustomerQueueInternal(ctx, rdb, customer, false, time.Time{})
}

func EnqueueCustomerQueueWithCapacity(ctx context.Context, rdb *redis.Client, customer CustomerQueue, day time.Time) (string, error) {
	return enqueueCustomerQueueInternal(ctx, rdb, customer, true, day)
}

func enqueueCustomerQueueInternal(ctx context.Context, rdb *redis.Client, customer CustomerQueue, checkCapacity bool, day time.Time) (string, error) {
	q := customer.Queue
	payload, err := json.Marshal(customer)
	if err != nil {
		return "", err
	}
	waitingKey := NormalWaitingQueueKey(q.BusinessID.String())
	if q.Priority {
		waitingKey = PriorityWaitingQueueKey(q.BusinessID.String())
	}
	keys := []string{
		GuestPhoneKey(q.BusinessID.String(), customer.PhoneNumber),
		ActiveQueueOwnerKey(q.BusinessID.String(), q.UserID.String()),
		CustomerQueueKey(q.ID.String()), QueueNameKey(q.ID.String()),
		waitingKey, config.StreamName,
	}
	capFlag := "0"
	if checkCapacity {
		capFlag = "1"
		keys = append(keys, DailyBusinessCapacityKey(q.BusinessID.String(), day))
	}
	return enqueueCustomer.Run(ctx, rdb, keys,
		q.ID.String(), q.BusinessID.String(), q.UserID.String(), q.Name, q.Priority,
		string(q.State), q.CreatedAt.Format(time.RFC3339Nano), string(payload), QueueScore(q),
		int(config.GuestExpirationTime.Seconds()), capFlag, config.QueueStreamMaxLen,
	).Text()
}

func GetCustomerQueue(ctx context.Context, rdb *redis.Client, queueID string) (*CustomerQueue, error) {
	payload, err := rdb.Get(ctx, CustomerQueueKey(queueID)).Bytes()
	if err != nil {
		return nil, err
	}
	var customer CustomerQueue
	if err := json.Unmarshal(payload, &customer); err != nil {
		return nil, err
	}
	return &customer, nil
}

// Compare-and-delete keeps delayed cleanup from removing a newer reservation.
var releaseCustomer = redis.NewScript(`
for _,key in ipairs(KEYS) do
  if redis.call('GET', key) == ARGV[1] then redis.call('DEL', key) end
end
return 1
`)

func ReleaseCustomerQueue(ctx context.Context, rdb *redis.Client, q domain.Queue) error {
	if rdb == nil {
		return nil
	}
	customer, err := GetCustomerQueue(ctx, rdb, q.ID.String())
	if err == redis.Nil {
		if q.UserID == nil {
			return nil
		}
		legacy, legacyErr := GetGuestQueueUser(ctx, rdb, q.UserID.String())
		if legacyErr == redis.Nil {
			return nil
		} // internal queue with no phone mapping
		if legacyErr != nil {
			return legacyErr
		}
		if legacy.QueueID != q.ID.String() || legacy.BusinessID != q.BusinessID.String() {
			return nil
		}
		return releaseCustomer.Run(ctx, rdb, []string{GuestPhoneKey(legacy.BusinessID, legacy.PhoneNumber)}, legacy.GuestID).Err()
	}
	if err != nil {
		return err
	}
	return releaseCustomer.Run(ctx, rdb, []string{
		GuestPhoneKey(customer.Queue.BusinessID.String(), customer.PhoneNumber),
		ActiveQueueOwnerKey(customer.Queue.BusinessID.String(), customer.Queue.UserID.String()),
	}, q.ID.String()).Err()
}

func SyncCustomerQueue(ctx context.Context, rdb *redis.Client, q domain.Queue) error {
	if q.IsTerminal() {
		return ReleaseCustomerQueue(ctx, rdb, q)
	}
	return nil
}
