package app

import (
	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/config"
	"QueueLite/internal/queue/domain"
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func QueueFromStreamValues(values map[string]any) (*domain.Queue, error) {
	id, err := requiredStreamUUID(values, "id", "queue id")
	if err != nil {
		return nil, err
	}
	businessID, err := requiredStreamUUID(values, "business_id", "business id")
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
	priority, err := strconv.ParseBool(priorityRaw)
	if err != nil {
		return nil, fmt.Errorf("parse priority: %w", err)
	}

	state := domain.QueueStateWaiting
	if stateRaw, ok := streamString(values, "state"); ok && stateRaw != "" {
		state = domain.QueueState(stateRaw)
	}

	userID, err := optionalStreamUUID(values, "user_id", "user id")
	if err != nil {
		return nil, err
	}

	createdAt, err := optionalStreamTime(values, "created_at", time.Now())
	if err != nil {
		return nil, err
	}

	return &domain.Queue{
		ID:         id,
		BusinessID: businessID,
		UserID:     userID,
		Name:       name,
		State:      state,
		Priority:   priority,
		CreatedAt:  createdAt,
	}, nil
}

func requiredStreamUUID(values map[string]any, key, label string) (uuid.UUID, error) {
	raw, err := requiredStreamString(values, key)
	if err != nil {
		return uuid.Nil, err
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse %s: %w", label, err)
	}
	return parsed, nil
}

func optionalStreamUUID(values map[string]any, key, label string) (*uuid.UUID, error) {
	raw, ok := streamString(values, key)
	if !ok || raw == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", label, err)
	}
	return &parsed, nil
}

func optionalStreamTime(values map[string]any, key string, fallback time.Time) (time.Time, error) {
	raw, ok := streamString(values, key)
	if !ok || raw == "" {
		return fallback, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
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

type BusinessCapacitySyncer interface {
	ConsumeBusinessCapacityBatch(ctx context.Context, businessID uuid.UUID, amount int) error
}

type DatabaseWorkerOptions struct {
	WorkerCount     int
	BatchSize       int64
	ReclaimIdle     time.Duration
	ReclaimInterval time.Duration
	RetryDelay      time.Duration
	MaxDeliveries   int64
	CapacitySyncer  BusinessCapacitySyncer
}

func DefaultDatabaseWorkerOptions() DatabaseWorkerOptions {
	return DatabaseWorkerOptions{
		WorkerCount:     2,
		BatchSize:       50,
		ReclaimIdle:     30 * time.Second,
		ReclaimInterval: 10 * time.Second,
		RetryDelay:      2 * time.Second,
		MaxDeliveries:   5,
	}
}

func (opts DatabaseWorkerOptions) withDefaults() DatabaseWorkerOptions {
	defaults := DefaultDatabaseWorkerOptions()
	if opts.WorkerCount <= 0 {
		opts.WorkerCount = defaults.WorkerCount
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = defaults.BatchSize
	}
	if opts.ReclaimIdle < 0 {
		opts.ReclaimIdle = defaults.ReclaimIdle
	}
	if opts.ReclaimInterval <= 0 {
		opts.ReclaimInterval = defaults.ReclaimInterval
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = defaults.RetryDelay
	}
	if opts.MaxDeliveries <= 0 {
		opts.MaxDeliveries = defaults.MaxDeliveries
	}
	return opts
}

func RunDatabaseWorkerStream(ctx context.Context, rdt *redis.Client, repo QueueRepo) {
	opts := DefaultDatabaseWorkerOptions()
	opts.WorkerCount = 1
	RunDatabaseWorkerPool(ctx, rdt, repo, opts)
}

func RunDatabaseWorkerPool(ctx context.Context, rdt *redis.Client, repo QueueRepo, opts DatabaseWorkerOptions) {
	if rdt == nil || repo == nil {
		return
	}
	opts = opts.withDefaults()

	if err := cache.EnsureQueueConsumerGroup(ctx, rdt); err != nil {
		log.Printf("[WORKER ERROR] failed to ensure queue consumer group: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < opts.WorkerCount; i++ {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			runSingleDatabaseWorker(ctx, rdt, repo, opts, workerIdx)
		}(i + 1)
	}
	wg.Wait()
}

func runDatabaseWorkerStream(ctx context.Context, rdt *redis.Client, repo QueueRepo, reclaimIdle, retryDelay time.Duration) {
	opts := DefaultDatabaseWorkerOptions()
	opts.WorkerCount = 1
	opts.BatchSize = 10
	opts.ReclaimIdle = reclaimIdle
	opts.ReclaimInterval = reclaimIdle
	if opts.ReclaimInterval <= 0 {
		opts.ReclaimInterval = time.Millisecond
	}
	opts.RetryDelay = retryDelay
	runSingleDatabaseWorker(ctx, rdt, repo, opts, 1)
}

type pendingQueueMessage struct {
	msg    redis.XMessage
	record domain.Queue
}

func runSingleDatabaseWorker(ctx context.Context, rdt *redis.Client, repo QueueRepo, opts DatabaseWorkerOptions, workerIdx int) {
	if err := cache.EnsureQueueConsumerGroup(ctx, rdt); err != nil {
		log.Printf("[WORKER ERROR] failed to ensure queue consumer group: %v", err)
	}
	consumer := &cache.StreamConsumer{
		Stream:          config.StreamName,
		Group:           config.QueueConsumerGroup,
		Consumer:        fmt.Sprintf("%s-%d-%s", config.QueueConsumerName, workerIdx, uuid.NewString()),
		BatchSize:       opts.BatchSize,
		ReclaimIdle:     opts.ReclaimIdle,
		ReclaimInterval: opts.ReclaimInterval,
		Block:           2 * time.Second,
	}

	for ctx.Err() == nil {
		messages, err := consumer.NextBatch(ctx, rdt)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[WORKER ERROR] failed to read queue stream: %v", err)
			_ = cache.EnsureQueueConsumerGroup(ctx, rdt)
			if !workerPause(ctx, opts.RetryDelay) {
				return
			}
			continue
		}

		valid := parseStreamMessages(ctx, rdt, messages)
		if len(valid) == 0 {
			continue
		}

		if persistStreamBatch(ctx, rdt, repo, opts, valid) {
			continue
		}
		if !workerPause(ctx, opts.RetryDelay) {
			return
		}
	}
}

func parseStreamMessages(ctx context.Context, rdt *redis.Client, messages []redis.XMessage) []pendingQueueMessage {
	valid := make([]pendingQueueMessage, 0, len(messages))
	for _, message := range messages {
		record, err := QueueFromStreamValues(message.Values)
		if err != nil {
			log.Printf("[WORKER ERROR] invalid queue stream message %s: %v", message.ID, err)
			if dlqErr := cache.AddQueueDeadLetter(ctx, rdt, message.ID, err.Error(), message.Values); dlqErr != nil {
				log.Printf("[WORKER ERROR] failed to add queue dead letter %s: %v", message.ID, dlqErr)
				continue // Never ack before the invalid payload is safely recorded.
			}
			if ackErr := rdt.XAck(ctx, config.StreamName, config.QueueConsumerGroup, message.ID).Err(); ackErr != nil {
				log.Printf("[WORKER ERROR] failed to ack invalid queue stream message %s: %v", message.ID, ackErr)
			}
			continue
		}
		valid = append(valid, pendingQueueMessage{msg: message, record: *record})
	}
	return valid
}

func persistStreamBatch(ctx context.Context, rdt *redis.Client, repo QueueRepo, opts DatabaseWorkerOptions, valid []pendingQueueMessage) bool {
	batch := make([]domain.Queue, len(valid))
	messageIDs := make([]string, len(valid))
	for i, item := range valid {
		batch[i] = item.record
		messageIDs[i] = item.msg.ID
	}

	baseCtx := context.WithoutCancel(ctx)
	persistCtx, cancel := context.WithTimeout(baseCtx, 15*time.Second)
	batchErr := repo.CreateQueues(persistCtx, batch)
	if batchErr == nil {
		syncPersistedCapacity(persistCtx, opts.CapacitySyncer, batch)
		cancel()
		ackQueueMessages(baseCtx, rdt, messageIDs)
		return true
	}
	cancel()

	log.Printf("[WORKER ERROR] failed to create queue batch (%d items): %v", len(batch), batchErr)
	if len(valid) == 1 {
		deadLetterQueueDBError(baseCtx, rdt, valid[0].msg, batchErr, opts.MaxDeliveries)
		return false
	}

	persistMessagesIndividually(baseCtx, rdt, repo, opts, valid)
	return false
}

// persistMessagesIndividually isolates poison-pill records so healthy tickets in the same batch still persist.
func persistMessagesIndividually(baseCtx context.Context, rdt *redis.Client, repo QueueRepo, opts DatabaseWorkerOptions, valid []pendingQueueMessage) {
	var persisted []domain.Queue
	var ackIDs []string
	for _, item := range valid {
		singleCtx, singleCancel := context.WithTimeout(baseCtx, 10*time.Second)
		err := repo.CreateQueues(singleCtx, []domain.Queue{item.record})
		singleCancel()
		if err == nil {
			persisted = append(persisted, item.record)
			ackIDs = append(ackIDs, item.msg.ID)
			continue
		}
		deadLetterQueueDBError(baseCtx, rdt, item.msg, err, opts.MaxDeliveries)
	}
	if len(persisted) > 0 {
		syncCtx, syncCancel := context.WithTimeout(baseCtx, 10*time.Second)
		syncPersistedCapacity(syncCtx, opts.CapacitySyncer, persisted)
		syncCancel()
	}
	ackQueueMessages(baseCtx, rdt, ackIDs)
}

func ackQueueMessages(ctx context.Context, rdt *redis.Client, messageIDs []string) {
	if len(messageIDs) == 0 {
		return
	}
	if err := rdt.XAck(ctx, config.StreamName, config.QueueConsumerGroup, messageIDs...).Err(); err != nil {
		log.Printf("[WORKER ERROR] failed to ack queue stream messages: %v", err)
	}
}

func syncPersistedCapacity(ctx context.Context, syncer BusinessCapacitySyncer, persisted []domain.Queue) {
	if syncer == nil || len(persisted) == 0 {
		return
	}
	counts := make(map[uuid.UUID]int)
	for _, q := range persisted {
		counts[q.BusinessID]++
	}
	for businessID, count := range counts {
		if err := syncer.ConsumeBusinessCapacityBatch(ctx, businessID, count); err != nil {
			log.Printf("[WORKER WARN] failed to sync capacity batch for business=%s count=%d: %v", businessID, count, err)
		}
	}
}

func deadLetterQueueDBError(ctx context.Context, rdt *redis.Client, msg redis.XMessage, cause error, maxDeliveries int64) {
	moved, err := cache.DeadLetterIfExhausted(ctx, rdt, config.StreamName, config.QueueConsumerGroup, config.QueueDeadLetterStreamName, msg, "db_error: "+cause.Error(), maxDeliveries)
	if err != nil {
		log.Printf("[WORKER ERROR] dead-letter check failed for queue message %s: %v", msg.ID, err)
	} else if moved {
		log.Printf("[WORKER ERROR] moved poison-pill queue message %s to dead letter: %v", msg.ID, cause)
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
