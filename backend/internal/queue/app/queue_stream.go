package app

import (
	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/config"
	"QueueLite/internal/queue/domain"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func QueueFromStreamValues(values map[string]any) (*domain.Queue, error) {
	queueID, err := requiredStreamString(values, "id")
	if err != nil {
		return nil, err
	}
	businessID, err := requiredStreamString(values, "business_id")
	if err != nil {
		return nil, err
	}
	name, err := requiredStreamString(values, "name")
	if err != nil {
		return nil, err
	}
	priorityRaw, err := requiredStreamString(values, "priority")
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(queueID)
	if err != nil {
		return nil, fmt.Errorf("parse queue id: %w", err)
	}
	businessUUID, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("parse business id: %w", err)
	}
	priority, err := strconv.ParseBool(priorityRaw)
	if err != nil {
		return nil, fmt.Errorf("parse priority: %w", err)
	}

	state := domain.QueueStateWaiting
	if stateRaw, ok := streamString(values, "state"); ok && stateRaw != "" {
		state = domain.QueueState(stateRaw)
	}

	var userID *uuid.UUID
	if userRaw, ok := streamString(values, "user_id"); ok && userRaw != "" {
		parsedUserID, err := uuid.Parse(userRaw)
		if err != nil {
			return nil, fmt.Errorf("parse user id: %w", err)
		}
		userID = &parsedUserID
	}

	createdAt := time.Now()
	if createdAtRaw, ok := streamString(values, "created_at"); ok && createdAtRaw != "" {
		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		createdAt = parsedCreatedAt
	}

	return &domain.Queue{
		ID:         id,
		BusinessID: businessUUID,
		UserID:     userID,
		Name:       name,
		State:      state,
		Priority:   priority,
		CreatedAt:  createdAt,
	}, nil
}

func requiredStreamString(values map[string]any, key string) (string, error) {
	value, ok := streamString(values, key)
	if !ok || value == "" {
		return "", fmt.Errorf("missing %s", key)
	}
	return value, nil
}

func streamString(values map[string]any, key string) (string, bool) {
	value, ok := values[key]
	if !ok || value == nil {
		return "", false
	}
	return fmt.Sprint(value), true
}

func RunDatabaseWorkerStream(ctx context.Context, rdt *redis.Client, repo QueueRepo) {
	runDatabaseWorkerStream(ctx, rdt, repo, 30*time.Second, 2*time.Second)
}

func runDatabaseWorkerStream(ctx context.Context, rdt *redis.Client, repo QueueRepo, reclaimIdle, retryDelay time.Duration) {
	consumer := config.QueueConsumerName + "-" + uuid.NewString()
	if err := cache.EnsureQueueConsumerGroup(ctx, rdt); err != nil {
		log.Printf("[WORKER ERROR] failed to ensure queue consumer group: %v", err)
	}
	cursor := "0-0"
	for ctx.Err() == nil {
		// Recover failed writes and interrupted consumers. CreateQueues is
		// idempotent by queue UUID, so a lost acknowledgement is safe to replay.
		messages, next, err := rdt.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream: config.StreamName, Group: config.QueueConsumerGroup,
			Consumer: consumer, MinIdle: reclaimIdle, Start: cursor, Count: 10,
		}).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[WORKER ERROR] failed to reclaim queue stream: %v", err)
			// A Redis restart may remove the group; recreate it before retrying.
			_ = cache.EnsureQueueConsumerGroup(ctx, rdt)
			if !workerPause(ctx, retryDelay) {
				return
			}
			continue
		}
		cursor = next
		if len(messages) == 0 {
			streams, err := rdt.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    config.QueueConsumerGroup,
				Consumer: consumer,
				Streams:  []string{config.StreamName, ">"},
				Count:    10,
				Block:    2 * time.Second,
			}).Result()

			// if the timer hits 2 second
			if errors.Is(err, redis.Nil) {
				continue
			} else if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[WORKER ERROR] Error reading stream: %v", err)
				if !workerPause(ctx, retryDelay) {
					return
				}
				continue
			}
			for _, stream := range streams {
				messages = append(messages, stream.Messages...)
			}
		}

		var batch []domain.Queue
		var messageIDs []string

		for _, message := range messages {
			record, err := QueueFromStreamValues(message.Values)
			if err != nil {
				log.Printf("[WORKER ERROR] invalid queue stream message %s: %v", message.ID, err)
				if deadLetterErr := cache.AddQueueDeadLetter(ctx, rdt, message.ID, err.Error(), message.Values); deadLetterErr != nil {
					log.Printf("[WORKER ERROR] failed to add queue dead letter %s: %v", message.ID, deadLetterErr)
					continue // Never ack before the invalid payload is safely recorded.
				}
				if ackErr := rdt.XAck(ctx, config.StreamName, config.QueueConsumerGroup, message.ID).Err(); ackErr != nil {
					log.Printf("[WORKER ERROR] failed to ack invalid queue stream message %s: %v", message.ID, ackErr)
				}
				continue
			}

			batch = append(batch, *record)
			messageIDs = append(messageIDs, message.ID)
		}

		if len(batch) == 0 {
			continue
		}

		if err := repo.CreateQueues(ctx, batch); err != nil {
			log.Printf("[WORKER ERROR] failed to create queue batch: %v", err)
			if !workerPause(ctx, retryDelay) {
				return
			}
			continue
		}

		if len(messageIDs) > 0 {
			if err := rdt.XAck(ctx, config.StreamName, config.QueueConsumerGroup, messageIDs...).Err(); err != nil {
				log.Printf("[WORKER ERROR] failed to ack queue stream batch: %v", err)
			}
		}
	}
}

func workerPause(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
