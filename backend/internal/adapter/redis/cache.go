package cache

import (
	"QueueLite/internal/queue/domain"
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func AddQueue(ctx context.Context, queueKey string, rdt *redis.Client, record domain.Queue) error {
	score := float64(record.CreatedAt.UnixMilli())
	if record.Priority {
		score -= 1_000_000_000_000
	}

	if err := rdt.ZAdd(ctx, queueKey, redis.Z{
		Score:  score,
		Member: record.ID.String(),
	}).Err(); err != nil {
		return fmt.Errorf("add queue to redis sorted set: %w", err)
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
