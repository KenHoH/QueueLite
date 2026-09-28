package cache

import (
	"QueueLite/internal/config"
	"QueueLite/internal/queue/domain"
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const priorityScoreOffset = 1_000_000_000_000

type GuestQueueUser struct {
	GuestID     string `json:"guestId"`
	BusinessID  string `json:"businessId"`
	QueueID     string `json:"queueId"`
	Username    string `json:"username"`
	PhoneNumber string `json:"phoneNumber"`
}

func WaitingQueueKey(businessID string) string {
	return fmt.Sprintf("queue:waiting:%s", businessID)
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
	return AddQueue(ctx, rdt, WaitingQueueKey(record.BusinessID.String()), record)
}

func RemoveWaitingQueue(ctx context.Context, rdt *redis.Client, businessID string, queueID string) error {
	if err := rdt.ZRem(ctx, WaitingQueueKey(businessID), queueID).Err(); err != nil {
		return fmt.Errorf("remove queue from redis sorted set: %w", err)
	}
	return nil
}

func QueueScore(record domain.Queue) float64 {
	score := float64(record.CreatedAt.UnixMilli())
	if record.Priority {
		score -= priorityScoreOffset
	}
	return score
}

func PopTopWaitingQueue(ctx context.Context, rdt *redis.Client, businessID string) (*string, error) {
	return GetTopQueue(ctx, WaitingQueueKey(businessID), rdt)
}

func PopTopWaitingQueueWithScore(ctx context.Context, rdt *redis.Client, businessID string) (*redis.Z, error) {
	results, err := rdt.ZPopMin(ctx, WaitingQueueKey(businessID), 1).Result()
	if err != nil {
		return nil, fmt.Errorf("pop top waiting queue from redis sorted set: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}
	return &results[0], nil
}

func RestoreWaitingQueue(ctx context.Context, rdt *redis.Client, businessID string, item redis.Z) error {
	if err := rdt.ZAdd(ctx, WaitingQueueKey(businessID), item).Err(); err != nil {
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
