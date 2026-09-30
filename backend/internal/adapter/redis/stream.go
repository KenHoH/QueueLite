package cache

import (
	"QueueLite/internal/config"
	"QueueLite/internal/queue/domain"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func EnsureQueueConsumerGroup(ctx context.Context, rdt *redis.Client) error {
	err := rdt.XGroupCreateMkStream(ctx, config.StreamName, config.QueueConsumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create queue consumer group: %w", err)
	}
	return nil
}

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

func AddQueueDeadLetter(ctx context.Context, rdt *redis.Client, messageID string, reason string, values map[string]any) error {
	deadLetterValues := map[string]any{
		"message_id": messageID,
		"reason":     reason,
	}
	for key, value := range values {
		deadLetterValues[key] = value
	}

	_, err := rdt.XAdd(ctx, &redis.XAddArgs{
		Stream: config.QueueDeadLetterStreamName,
		MaxLen: 1000,
		Approx: true,
		ID:     "*",
		Values: deadLetterValues,
	}).Result()
	if err != nil {
		return fmt.Errorf("add queue dead letter: %w", err)
	}
	return nil
}
