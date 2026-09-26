package cache

import (
	"QueueLite/internal/config"
	"QueueLite/internal/queue/domain"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func AddQueueStream(ctx context.Context, rdt *redis.Client, record domain.Queue) error {
	values := map[string]any{
		"id":          record.ID.String(),
		"business_id": record.BusinessID.String(),
		"name":        record.Name,
		"priority":    record.Priority,
		"state":       string(record.State),
		"created_at":  record.CreatedAt.Format(time.RFC3339Nano),
	}
	if record.UserID != nil {
		values["user_id"] = record.UserID.String()
	}

	_, err := rdt.XAdd(ctx, &redis.XAddArgs{
		Stream: config.StreamName,
		MaxLen: 1000,
		Approx: true,
		ID:     "*", // Auto-generate message ID
		Values: values,
	}).Result()
	if err != nil {
		return fmt.Errorf("add queue stream: %w", err)
	}
	return nil
}
