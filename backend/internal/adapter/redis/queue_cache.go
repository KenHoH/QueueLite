package cache

import (
	"QueueLite/internal/config"
	"QueueLite/internal/queue/domain"
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const maxPriorityStreak = 5

type GuestQueueUser struct {
	GuestID     string `json:"guestId"`
	BusinessID  string `json:"businessId"`
	QueueID     string `json:"queueId"`
	Username    string `json:"username"`
	PhoneNumber string `json:"phoneNumber"`
}

type WaitingQueueItem struct {
	redis.Z
	QueueKey string
}

func WaitingQueueKey(businessID string) string {
	return NormalWaitingQueueKey(businessID)
}

func PriorityWaitingQueueKey(businessID string) string {
	return fmt.Sprintf("queue:waiting:priority:%s", businessID)
}

func NormalWaitingQueueKey(businessID string) string {
	return fmt.Sprintf("queue:waiting:normal:%s", businessID)
}

func LegacyWaitingQueueKey(businessID string) string {
	return fmt.Sprintf("queue:waiting:%s", businessID)
}

func PriorityStreakKey(businessID string) string {
	return fmt.Sprintf("queue:priority_streak:%s", businessID)
}

func QueueEventChannel(businessID string) string {
	return fmt.Sprintf("queue:events:%s", businessID)
}

func QueueNameKey(queueID string) string {
	return fmt.Sprintf("queue:name:%s", queueID)
}

func GuestKey(guestID string) string {
	return fmt.Sprintf("queue:guest:%s", guestID)
}

func GuestPhoneKey(businessID string, phoneNumber string) string {
	return fmt.Sprintf("queue:guest_phone:%s:%s", businessID, phoneNumber)
}

func GetQueueName(ctx context.Context, rdt *redis.Client, queueID string) (string, error) {
	val, err := rdt.Get(ctx, QueueNameKey(queueID)).Bytes()
	if err != nil {
		return "", err
	}
	return string(val), nil
}

func SetQueueName(ctx context.Context, rdt *redis.Client, queueID string, queueName string) error {
	return rdt.Set(ctx, QueueNameKey(queueID), queueName, config.QueueExpirationTime).Err()
}

func QueueExists(ctx context.Context, rdt *redis.Client, queueID string) (bool, error) {
	count, err := rdt.Exists(ctx, QueueNameKey(queueID)).Result()
	if err != nil {
		return false, fmt.Errorf("check queue exists in redis: %w", err)
	}
	return count > 0, nil
}

func ReserveQueuePhone(ctx context.Context, rdt *redis.Client, businessID string, phoneNumber string, ownerID string) (bool, error) {
	ok, err := rdt.SetNX(ctx, GuestPhoneKey(businessID, phoneNumber), ownerID, config.GuestExpirationTime).Result()
	if err != nil {
		return false, fmt.Errorf("reserve queue phone: %w", err)
	}
	return ok, nil
}

func ReleaseQueuePhone(ctx context.Context, rdt *redis.Client, businessID string, phoneNumber string) error {
	return rdt.Del(ctx, GuestPhoneKey(businessID, phoneNumber)).Err()
}

func SetGuestQueueUser(ctx context.Context, rdt *redis.Client, guest GuestQueueUser) error {
	payload, err := json.Marshal(guest)
	if err != nil {
		return fmt.Errorf("marshal guest queue user: %w", err)
	}
	return rdt.Set(ctx, GuestKey(guest.GuestID), payload, config.GuestExpirationTime).Err()
}

func AddQueue(ctx context.Context, rdt *redis.Client, queueKey string, record domain.Queue) error {
	score := QueueScore(record)

	err := SetQueueName(ctx, rdt, record.ID.String(), record.Name)
	if err != nil {
		return err
	}

	if err = rdt.ZAdd(ctx, queueKey, redis.Z{
		Score:  score,
		Member: record.ID.String(),
	}).Err(); err != nil {
		return fmt.Errorf("add queue to redis sorted set: %w", err)
	}

	return nil
}

func AddWaitingQueue(ctx context.Context, rdt *redis.Client, record domain.Queue) error {
	queueKey := NormalWaitingQueueKey(record.BusinessID.String())
	if record.Priority {
		queueKey = PriorityWaitingQueueKey(record.BusinessID.String())
	}
	return AddQueue(ctx, rdt, queueKey, record)
}

func RemoveWaitingQueue(ctx context.Context, rdt *redis.Client, businessID string, queueID string) error {
	pipe := rdt.TxPipeline()
	pipe.ZRem(ctx, PriorityWaitingQueueKey(businessID), queueID)
	pipe.ZRem(ctx, NormalWaitingQueueKey(businessID), queueID)
	pipe.ZRem(ctx, LegacyWaitingQueueKey(businessID), queueID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("remove queue from redis sorted set: %w", err)
	}
	return nil
}

func QueueScore(record domain.Queue) float64 {
	return float64(record.CreatedAt.UnixMilli())
}

func PopTopWaitingQueue(ctx context.Context, rdt *redis.Client, businessID string) (*string, error) {
	return GetTopQueue(ctx, WaitingQueueKey(businessID), rdt)
}

func PopTopWaitingQueueWithScore(ctx context.Context, rdt *redis.Client, businessID string) (*redis.Z, error) {
	item, err := popTopWaitingQueueFromKey(ctx, rdt, WaitingQueueKey(businessID))
	if item == nil || err != nil {
		return item, err
	}
	return item, nil
}

func PopNextFairWaitingQueueWithScore(ctx context.Context, rdt *redis.Client, businessID string) (*WaitingQueueItem, error) {
	streak, err := rdt.Get(ctx, PriorityStreakKey(businessID)).Int()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("get priority streak: %w", err)
	}

	priorityKey := PriorityWaitingQueueKey(businessID)

	if streak < maxPriorityStreak {
		item, err := popTopWaitingQueueFromKey(ctx, rdt, priorityKey)
		if err != nil {
			return nil, err
		}
		if item != nil {
			if err := rdt.Set(ctx, PriorityStreakKey(businessID), streak+1, 0).Err(); err != nil {
				return nil, fmt.Errorf("set priority streak: %w", err)
			}
			return &WaitingQueueItem{Z: *item, QueueKey: priorityKey}, nil
		}

		normalItem, err := popNormalWaitingQueue(ctx, rdt, businessID)
		if err != nil {
			return nil, err
		}
		if normalItem != nil {
			if err := rdt.Set(ctx, PriorityStreakKey(businessID), 0, 0).Err(); err != nil {
				return nil, fmt.Errorf("reset priority streak: %w", err)
			}
			return normalItem, nil
		}
		return nil, nil
	}

	normalItem, err := popNormalWaitingQueue(ctx, rdt, businessID)
	if err != nil {
		return nil, err
	}
	if normalItem != nil {
		if err := rdt.Set(ctx, PriorityStreakKey(businessID), 0, 0).Err(); err != nil {
			return nil, fmt.Errorf("reset priority streak: %w", err)
		}
		return normalItem, nil
	}

	item, err := popTopWaitingQueueFromKey(ctx, rdt, priorityKey)
	if err != nil {
		return nil, err
	}
	if item != nil {
		if err := rdt.Set(ctx, PriorityStreakKey(businessID), maxPriorityStreak, 0).Err(); err != nil {
			return nil, fmt.Errorf("keep priority streak capped: %w", err)
		}
		return &WaitingQueueItem{Z: *item, QueueKey: priorityKey}, nil
	}
	return nil, nil
}

func popTopWaitingQueueFromKey(ctx context.Context, rdt *redis.Client, queueKey string) (*redis.Z, error) {
	results, err := rdt.ZPopMin(ctx, queueKey, 1).Result()
	if err != nil {
		return nil, fmt.Errorf("pop top waiting queue from redis sorted set: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}
	return &results[0], nil
}

func popNormalWaitingQueue(ctx context.Context, rdt *redis.Client, businessID string) (*WaitingQueueItem, error) {
	normalKey := NormalWaitingQueueKey(businessID)
	item, err := popTopWaitingQueueFromKey(ctx, rdt, normalKey)
	if err != nil {
		return nil, err
	}
	if item != nil {
		return &WaitingQueueItem{Z: *item, QueueKey: normalKey}, nil
	}

	legacyKey := LegacyWaitingQueueKey(businessID)
	item, err = popTopWaitingQueueFromKey(ctx, rdt, legacyKey)
	if err != nil {
		return nil, err
	}
	if item != nil {
		return &WaitingQueueItem{Z: *item, QueueKey: legacyKey}, nil
	}
	return nil, nil
}

func RestoreWaitingQueue(ctx context.Context, rdt *redis.Client, businessID string, item redis.Z) error {
	if err := rdt.ZAdd(ctx, WaitingQueueKey(businessID), item).Err(); err != nil {
		return fmt.Errorf("restore waiting queue to redis sorted set: %w", err)
	}
	return nil
}

func RestoreWaitingQueueItem(ctx context.Context, rdt *redis.Client, item WaitingQueueItem) error {
	if err := rdt.ZAdd(ctx, item.QueueKey, item.Z).Err(); err != nil {
		return fmt.Errorf("restore waiting queue to redis sorted set: %w", err)
	}
	return nil
}

func GetTopQueue(ctx context.Context, queueKey string, rdt *redis.Client) (*string, error) {
	results, err := rdt.ZPopMin(ctx, queueKey, 1).Result()
	if err != nil {
		return nil, fmt.Errorf("pop top queue from redis sorted set: %w", err)
	}

	// there's nothing to get
	if len(results) == 0 {
		return nil, nil
	}

	payload := results[0]
	memberStr, ok := payload.Member.(string)
	if !ok {
		return nil, fmt.Errorf("redis sorted set member is not a string")
	}
	return &memberStr, nil
}

func GetQueue(ctx context.Context, queueKey string, rdt *redis.Client) ([]redis.Z, error) {
	tasks, err := rdt.ZRangeWithScores(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func GetWaitingQueues(ctx context.Context, businessID string, rdt *redis.Client) ([]redis.Z, error) {
	priorityTasks, err := GetQueue(ctx, PriorityWaitingQueueKey(businessID), rdt)
	if err != nil {
		return nil, err
	}
	normalTasks, err := GetQueue(ctx, NormalWaitingQueueKey(businessID), rdt)
	if err != nil {
		return nil, err
	}
	legacyTasks, err := GetQueue(ctx, LegacyWaitingQueueKey(businessID), rdt)
	if err != nil {
		return nil, err
	}

	tasks := make([]redis.Z, 0, len(priorityTasks)+len(normalTasks)+len(legacyTasks))
	tasks = append(tasks, priorityTasks...)
	tasks = append(tasks, normalTasks...)
	tasks = append(tasks, legacyTasks...)
	return tasks, nil
}
