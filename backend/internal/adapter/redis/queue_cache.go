package cache

import (
	"QueueLite/internal/config"
	"QueueLite/internal/queue/domain"
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const priorityScoreOffset = 1_000_000_000_000

func WaitingQueueKey(businessID string) string {
	return fmt.Sprintf("queue:waiting:%s", businessID)
}

func QueueEventChannel(businessID string) string {
	return fmt.Sprintf("queue:events:%s", businessID)
}

func QueueNameKey(queueID string) string {
	return fmt.Sprintf("queue:name:%s", queueID)
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
