package app

import (
	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	"QueueLite/internal/config"
	"QueueLite/internal/queue/domain"
	subscriptionapp "QueueLite/internal/subscription/app"
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type QueueService struct {
	repo                QueueRepo
	subscriptionService *subscriptionapp.SubscriptionService
	rdt                 *redis.Client
}

func NewQueueService(repo QueueRepo, subscriptionService *subscriptionapp.SubscriptionService, redis *redis.Client) *QueueService {
	service := &QueueService{
		repo:                repo,
		subscriptionService: subscriptionService,
		rdt:                 redis,
	}
	return service
}

func (s *QueueService) RegisterQueue(ctx context.Context, queue domain.Queue) (*domain.Queue, error) {
	if queue.BusinessID == uuid.Nil {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
	}
	if queue.UserID == nil {
		return nil, apperror.New(apperror.KindInvalid, "QUEUE_OWNER_REQUIRED", "user id is required")
	}

	active, err := s.repo.GetActiveQueueByUserAndBusiness(ctx, *queue.UserID, queue.BusinessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_ACTIVE_QUEUE_ERROR", "failed to check active queue", err)
	}
	if active != nil {
		return nil, apperror.New(apperror.KindInvalid, "ACTIVE_QUEUE_EXISTS", "user already has active queue in this business")
	}

	if queue.State == "" {
		queue.State = domain.QueueStateWaiting
	}
	if strings.TrimSpace(queue.Name) == "" {
		name, err := s.repo.GenerateDailyQueueName(ctx, queue.BusinessID, time.Now())
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "GENERATE_QUEUE_NAME_ERROR", "failed to generate queue name", err)
		}
		queue.Name = name
	}

	if queue.ID == uuid.Nil {
		queue.ID = uuid.New()
	}
	if queue.CreatedAt.IsZero() {
		queue.CreatedAt = time.Now()
	}

	if err := cache.AddQueueStream(ctx, s.rdt, queue); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "REGISTER_QUEUE_STREAM_ERROR", "failed to queue registration for database worker", err)
	}

	return &queue, nil
}

func (s *QueueService) GetQueue(ctx context.Context, id uuid.UUID) (*domain.Queue, error) {
	queue, err := s.repo.GetQueue(ctx, id)
	if err != nil {
		if errors.Is(err, ErrQueueNotFound) {
			return nil, apperror.Wrap(apperror.KindNotFound, "QUEUE_NOT_FOUND", "queue not found", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "GET_QUEUE_ERROR", "failed to get queue", err)
	}
	return queue, nil
}

func (s *QueueService) GetQueueState(ctx context.Context, queueID uuid.UUID) (domain.QueueState, error) {
	queue, err := s.GetQueue(ctx, queueID)
	if err != nil {
		return "", err
	}
	return queue.State, nil
}

func (s *QueueService) GetAllQueueByBusiness(ctx context.Context, businessID uuid.UUID) ([]domain.Queue, error) {
	queues, err := s.repo.GetAllQueueByBusiness(ctx, businessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_QUEUES_ERROR", "failed to get business queues", err)
	}
	return queues, nil
}

func (s *QueueService) GetAllQueueByBusinessFilterState(ctx context.Context, businessID uuid.UUID, state domain.QueueState) ([]domain.Queue, error) {
	queues, err := s.repo.GetAllQueueByBusinessFilterState(ctx, businessID, state)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_QUEUES_BY_STATE_ERROR", "failed to get business queues by state", err)
	}
	return queues, nil
}

func (s *QueueService) GetBusinessPublicQueueSummary(ctx context.Context, businessID uuid.UUID) (*domain.PublicQueueSummary, error) {
	summary, err := s.repo.GetBusinessPublicQueueSummary(ctx, businessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_PUBLIC_QUEUE_SUMMARY_ERROR", "failed to get public queue summary", err)
	}
	return summary, nil
}

func (s *QueueService) UpdateState(ctx context.Context, queueID uuid.UUID, state domain.QueueState) error {
	queue, err := s.GetQueue(ctx, queueID)
	if err != nil {
		return err
	}
	queue.State = state
	if err := s.repo.UpdateQueue(ctx, queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "UPDATE_QUEUE_STATE_ERROR", "failed to update queue state", err)
	}
	return nil
}

func (s *QueueService) MarkAsDone(ctx context.Context, queueID uuid.UUID) error {
	return s.UpdateState(ctx, queueID, domain.QueueStateCompleted)
}

func (s *QueueService) MarkAsProcessing(ctx context.Context, queueID uuid.UUID) error {
	return s.UpdateState(ctx, queueID, domain.QueueStateProcessing)
}

func (s *QueueService) MarkAsCancelled(ctx context.Context, queueID uuid.UUID) error {
	return s.UpdateState(ctx, queueID, domain.QueueStateCancelled)
}

func (s *QueueService) MarkAsSkipped(ctx context.Context, queueID uuid.UUID) error {
	return s.UpdateState(ctx, queueID, domain.QueueStateSkipped)
}

func (s *QueueService) UpdateQueue(ctx context.Context, queue domain.Queue) error {
	if queue.Name == "" {
		return apperror.New(apperror.KindInvalid, "QUEUE_NAME_REQUIRED", "missing queue name")
	}
	if queue.State == "" {
		queue.State = domain.QueueStateWaiting
	}
	if err := s.repo.UpdateQueue(ctx, &queue); err != nil {
		if errors.Is(err, ErrQueueNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "QUEUE_NOT_FOUND", "queue not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "UPDATE_QUEUE_ERROR", "failed to update queue", err)
	}
	return nil
}

func (s *QueueService) DeleteQueue(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteQueue(ctx, id); err != nil {
		if errors.Is(err, ErrQueueNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "QUEUE_NOT_FOUND", "queue not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "DELETE_QUEUE_ERROR", "failed to delete queue", err)
	}
	return nil
}

func RunDatabaseWorkerStream(ctx context.Context, rdt *redis.Client, repo QueueRepo) {
	lastSeenID := "0"

	for {
		streams, err := rdt.XRead(ctx, &redis.XReadArgs{
			Streams: []string{config.StreamName, lastSeenID},
			Count:   10,              // max total events that enter the stream to be processed
			Block:   2 * time.Second, // max idle time no events in
		}).Result()

		// if the timer hits 2 second
		if errors.Is(err, redis.Nil) {
			continue
		} else if err != nil {
			log.Printf("[WORKER ERROR] Error reading stream: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		var batch []domain.Queue
		lastBatchID := ""

		for _, stream := range streams {
			for _, message := range stream.Messages {
				record, err := queueFromStreamValues(message.Values)
				if err != nil {
					log.Printf("[WORKER ERROR] invalid queue stream message %s: %v", message.ID, err)
					lastSeenID = message.ID
					continue
				}

				batch = append(batch, *record)
				lastBatchID = message.ID
			}
		}

		if len(batch) == 0 {
			continue
		}

		if err := repo.CreateQueues(ctx, batch); err != nil {
			log.Printf("[WORKER ERROR] failed to create queue batch: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for i := range batch {
			queueKey := batch[i].BusinessID.String()
			if err := cache.AddQueue(ctx, queueKey, rdt, batch[i]); err != nil {
				log.Printf("[WORKER ERROR] failed to add queue to redis: %v", err)
			}
			if err := cache.PublishMessage(ctx, rdt, queueKey, "UserJoinedQueue"); err != nil {
				log.Printf("[WORKER ERROR] failed to publish queue event: %v", err)
			}
		}

		if lastBatchID != "" {
			lastSeenID = lastBatchID
		}
	}
}
