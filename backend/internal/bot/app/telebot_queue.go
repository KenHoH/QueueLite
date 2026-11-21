package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/config"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	telebotJobTypeUserMessage  = "user_message"
	telebotJobTypePhoneRequest = "phone_request"
)

// EnqueueUserNotification queues a Telegram notification asynchronously via Redis Stream
// so counter workflows are never blocked by external Telegram API latency or outages.
func (s *TelebotService) EnqueueUserNotification(ctx context.Context, userID uuid.UUID, text string) error {
	if s == nil || s.rdb == nil || userID == uuid.Nil || strings.TrimSpace(text) == "" || strings.TrimSpace(config.LoadTelebotAPIKey()) == "" {
		return nil
	}
	return s.enqueueJob(ctx, map[string]any{
		"type":    telebotJobTypeUserMessage,
		"user_id": userID.String(),
		"text":    text,
	})
}

// EnqueuePhoneRequest queues a Telegram phone-number prompt asynchronously via Redis Stream.
func (s *TelebotService) EnqueuePhoneRequest(ctx context.Context, chatID string) error {
	chatID = strings.TrimSpace(chatID)
	if s == nil || chatID == "" {
		return nil
	}
	if s.rdb == nil {
		return s.requestPhonenumber(ctx, chatID)
	}
	return s.enqueueJob(ctx, map[string]any{
		"type":    telebotJobTypePhoneRequest,
		"chat_id": chatID,
	})
}

func (s *TelebotService) enqueueJob(ctx context.Context, values map[string]any) error {
	_, err := s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: config.TelebotNotificationStreamName,
		MaxLen: 10000,
		Approx: true,
		ID:     "*",
		Values: values,
	}).Result()
	if err != nil {
		return fmt.Errorf("enqueue telegram notification: %w", err)
	}
	return nil
}

// RunNotificationWorker consumes queued Telegram jobs asynchronously from Redis Stream.
func (s *TelebotService) RunNotificationWorker(ctx context.Context, rdb *redis.Client) {
	if s == nil || rdb == nil {
		return
	}
	if err := cache.EnsureConsumerGroup(ctx, rdb, config.TelebotNotificationStreamName, config.TelebotConsumerGroup); err != nil {
		log.Printf("[TELEBOT WORKER ERROR] failed to ensure consumer group: %v", err)
	}
	consumer := &cache.StreamConsumer{
		Stream:          config.TelebotNotificationStreamName,
		Group:           config.TelebotConsumerGroup,
		Consumer:        config.TelebotConsumerName + "-" + uuid.NewString(),
		BatchSize:       10,
		ReclaimIdle:     30 * time.Second,
		ReclaimInterval: 10 * time.Second,
		Block:           2 * time.Second,
	}

	const maxDeliveries int64 = 5
	for ctx.Err() == nil {
		messages, err := consumer.NextBatch(ctx, rdb)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[TELEBOT WORKER ERROR] failed to read notification stream: %v", err)
			_ = cache.EnsureConsumerGroup(ctx, rdb, config.TelebotNotificationStreamName, config.TelebotConsumerGroup)
			if !telebotWorkerPause(ctx, 2*time.Second) {
				return
			}
			continue
		}

		baseCtx := context.WithoutCancel(ctx)
		for _, msg := range messages {
			if err := s.processNotificationMessage(baseCtx, msg.Values); err != nil {
				log.Printf("[TELEBOT WORKER ERROR] failed to process message %s: %v", msg.ID, err)
				_, _ = cache.DeadLetterIfExhausted(baseCtx, rdb, config.TelebotNotificationStreamName, config.TelebotConsumerGroup, config.TelebotDeadLetterStreamName, msg, err.Error(), maxDeliveries)
				continue
			}
			if err := rdb.XAck(baseCtx, config.TelebotNotificationStreamName, config.TelebotConsumerGroup, msg.ID).Err(); err != nil {
				log.Printf("[TELEBOT WORKER ERROR] failed to ack message %s: %v", msg.ID, err)
			}
		}
	}
}

func streamFieldString(values map[string]any, key string) string {
	raw, ok := values[key]
	if !ok || raw == nil {
		return ""
	}
	return fmt.Sprint(raw)
}

func (s *TelebotService) processNotificationMessage(ctx context.Context, values map[string]any) error {
	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch jobType := streamFieldString(values, "type"); jobType {
	case telebotJobTypeUserMessage:
		userID, err := uuid.Parse(streamFieldString(values, "user_id"))
		if err != nil {
			log.Printf("[TELEBOT WORKER WARN] dropping notification with invalid user_id %q", values["user_id"])
			return nil
		}
		return s.SendMessageToUser(sendCtx, userID, streamFieldString(values, "text"))
	case telebotJobTypePhoneRequest:
		chatID := strings.TrimSpace(streamFieldString(values, "chat_id"))
		if chatID == "" {
			return nil
		}
		err := s.requestPhonenumber(sendCtx, chatID)
		if errors.Is(err, ErrTelebotNotConfigured) {
			return nil
		}
		return err
	default:
		log.Printf("[TELEBOT WORKER WARN] dropping unknown notification job type %q", jobType)
		return nil
	}
}

func telebotWorkerPause(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
