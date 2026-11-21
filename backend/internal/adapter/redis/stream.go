package cache

import (
	"QueueLite/internal/config"
	"QueueLite/internal/queue/domain"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func EnsureConsumerGroup(ctx context.Context, rdt *redis.Client, stream, group string) error {
	err := rdt.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create consumer group %s on %s: %w", group, stream, err)
	}
	return nil
}

func EnsureQueueConsumerGroup(ctx context.Context, rdt *redis.Client) error {
	return EnsureConsumerGroup(ctx, rdt, config.StreamName, config.QueueConsumerGroup)
}

type StreamConsumer struct {
	Stream          string
	Group           string
	Consumer        string
	BatchSize       int64
	ReclaimIdle     time.Duration
	ReclaimInterval time.Duration
	Block           time.Duration
	cursor          string
	lastReclaim     time.Time
}

// NextBatch periodically reclaims idle pending messages and otherwise blocks on new stream entries.
func (c *StreamConsumer) NextBatch(ctx context.Context, rdt *redis.Client) ([]redis.XMessage, error) {
	if c.cursor == "" {
		c.cursor = "0-0"
	}
	if c.Block <= 0 {
		c.Block = 2 * time.Second
	}
	if c.lastReclaim.IsZero() || c.cursor != "0-0" || time.Since(c.lastReclaim) >= c.ReclaimInterval {
		reclaimed, next, err := rdt.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   c.Stream,
			Group:    c.Group,
			Consumer: c.Consumer,
			MinIdle:  c.ReclaimIdle,
			Start:    c.cursor,
			Count:    c.BatchSize,
		}).Result()
		if err != nil {
			return nil, err
		}
		c.cursor = next
		if c.cursor == "0-0" {
			c.lastReclaim = time.Now()
		}
		if len(reclaimed) > 0 {
			return reclaimed, nil
		}
	}

	streams, err := rdt.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.Group,
		Consumer: c.Consumer,
		Streams:  []string{c.Stream, ">"},
		Count:    c.BatchSize,
		Block:    c.Block,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var messages []redis.XMessage
	for _, stream := range streams {
		messages = append(messages, stream.Messages...)
	}
	return messages, nil
}

// DeadLetterIfExhausted moves a repeatedly failing message to dlqStream and acknowledges it
// once its delivery count reaches maxDeliveries. Returns true when dead-lettered.
func DeadLetterIfExhausted(ctx context.Context, rdt *redis.Client, stream, group, dlqStream string, msg redis.XMessage, reason string, maxDeliveries int64) (bool, error) {
	if maxDeliveries <= 0 {
		return false, nil
	}
	pending, err := rdt.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream,
		Group:  group,
		Start:  msg.ID,
		End:    msg.ID,
		Count:  1,
	}).Result()
	if err != nil || len(pending) == 0 || pending[0].RetryCount < maxDeliveries {
		return false, err
	}
	if err := AddStreamDeadLetter(ctx, rdt, dlqStream, msg.ID, reason, msg.Values); err != nil {
		return false, err
	}
	if err := rdt.XAck(ctx, stream, group, msg.ID).Err(); err != nil {
		return true, err
	}
	return true, nil
}

func AddQueueStream(ctx context.Context, rdt *redis.Client, record domain.Queue) error {
	values := map[string]any{
		"id":          record.ID.String(),
		"business_id": record.BusinessID.String(),
		"name":        record.Name,
		"priority":    record.Priority,
		"state":       string(record.State),
		"created_at":  record.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if record.UserID != nil {
		values["user_id"] = record.UserID.String()
	}

	_, err := rdt.XAdd(ctx, &redis.XAddArgs{
		Stream: config.StreamName,
		MaxLen: config.QueueStreamMaxLen,
		Approx: true,
		ID:     "*", // Auto-generate message ID
		Values: values,
	}).Result()
	if err != nil {
		return fmt.Errorf("add queue stream: %w", err)
	}
	return nil
}

func AddStreamDeadLetter(ctx context.Context, rdt *redis.Client, dlqStream, messageID, reason string, values map[string]any) error {
	deadLetterValues := map[string]any{
		"message_id": messageID,
		"reason":     reason,
	}
	for key, value := range values {
		deadLetterValues[key] = value
	}

	_, err := rdt.XAdd(ctx, &redis.XAddArgs{
		Stream: dlqStream,
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

func AddQueueDeadLetter(ctx context.Context, rdt *redis.Client, messageID string, reason string, values map[string]any) error {
	return AddStreamDeadLetter(ctx, rdt, config.QueueDeadLetterStreamName, messageID, reason, values)
}
