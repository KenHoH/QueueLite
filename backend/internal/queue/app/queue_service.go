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

func (s *QueueService) SubscribeQueueChannel(ctx context.Context, businessID string) *redis.PubSub {
	return cache.SubscribeChannel(ctx, s.rdt, cache.QueueEventChannel(businessID))
}

func (s *QueueService) publishQueueUpdate(ctx context.Context, businessID uuid.UUID) error {
	return cache.PublishMessage(ctx, s.rdt, cache.QueueEventChannel(businessID.String()), "queue.update")
}

func (s *QueueService) syncQueueCache(ctx context.Context, queue domain.Queue) error {
	if queue.State == domain.QueueStateWaiting {
		return cache.AddWaitingQueue(ctx, s.rdt, queue)
	}
	return cache.RemoveWaitingQueue(ctx, s.rdt, queue.BusinessID.String(), queue.ID.String())
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

	if err := s.syncQueueCache(ctx, queue); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "CACHE_QUEUE_ERROR", "failed to register queue to redis", err)
	}

	if err := s.publishQueueUpdate(ctx, queue.BusinessID); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "PUBLISH_MESSAGE_ERROR", "failed to publish queue", err)
	}
	return &queue, nil
}

func (s *QueueService) GetQueueCache(ctx context.Context, businessID string) ([]redis.Z, error) {
	return cache.GetQueue(ctx, cache.WaitingQueueKey(businessID), s.rdt)
}

func (s *QueueService) GetQueueNameCache(ctx context.Context, queueID string) (string, error) {
	return cache.GetQueueName(ctx, s.rdt, queueID)
}

func (s *QueueService) GetWaitingQueueSnapshot(ctx context.Context, businessID uuid.UUID) ([]QueueSnapshotItem, error) {
	tasks, err := s.GetQueueCache(ctx, businessID.String())
	if err != nil {
		return nil, err
	}

	response := make([]QueueSnapshotItem, 0, len(tasks))
	for _, data := range tasks {
		queueID, ok := data.Member.(string)
		if !ok {
			continue
		}

		queueName, err := s.GetQueueNameCache(ctx, queueID)
		if err != nil {
			continue
		}

		response = append(response, QueueSnapshotItem{
			QueueID:   queueID,
			QueueName: queueName,
		})
	}
	return response, nil
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
	if err := s.syncQueueCache(ctx, *queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "UPDATE_QUEUE_STATE_CACHE_ERROR", "failed to update queue cache", err)
	}
	_ = s.publishQueueUpdate(ctx, queue.BusinessID)
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
	if err := s.syncQueueCache(ctx, queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "UPDATE_QUEUE_CACHE_ERROR", "failed to update queue cache", err)
	}
	_ = s.publishQueueUpdate(ctx, queue.BusinessID)
	return nil
}

func (s *QueueService) DeleteQueue(ctx context.Context, id uuid.UUID) error {
	queue, err := s.GetQueue(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteQueue(ctx, id); err != nil {
		if errors.Is(err, ErrQueueNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "QUEUE_NOT_FOUND", "queue not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "DELETE_QUEUE_ERROR", "failed to delete queue", err)
	}
	_ = cache.RemoveWaitingQueue(ctx, s.rdt, queue.BusinessID.String(), queue.ID.String())
	_ = s.publishQueueUpdate(ctx, queue.BusinessID)
	return nil
}

func RunDatabaseWorkerStream(ctx context.Context, rdt *redis.Client, repo QueueRepo) {
	if err := cache.EnsureQueueConsumerGroup(ctx, rdt); err != nil {
		log.Printf("[WORKER ERROR] failed to ensure queue consumer group: %v", err)
	}

	for {
		streams, err := rdt.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    config.QueueConsumerGroup,
			Consumer: config.QueueConsumerName,
			Streams:  []string{config.StreamName, ">"},
			Count:    10,
			Block:    2 * time.Second,
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
		var messageIDs []string

		for _, stream := range streams {
			for _, message := range stream.Messages {
				record, err := QueueFromStreamValues(message.Values)
				if err != nil {
					log.Printf("[WORKER ERROR] invalid queue stream message %s: %v", message.ID, err)
					if deadLetterErr := cache.AddQueueDeadLetter(ctx, rdt, message.ID, err.Error(), message.Values); deadLetterErr != nil {
						log.Printf("[WORKER ERROR] failed to add queue dead letter %s: %v", message.ID, deadLetterErr)
					}
					if ackErr := rdt.XAck(ctx, config.StreamName, config.QueueConsumerGroup, message.ID).Err(); ackErr != nil {
						log.Printf("[WORKER ERROR] failed to ack invalid queue stream message %s: %v", message.ID, ackErr)
					}
					continue
				}

				batch = append(batch, *record)
				messageIDs = append(messageIDs, message.ID)
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

		if len(messageIDs) > 0 {
			if err := rdt.XAck(ctx, config.StreamName, config.QueueConsumerGroup, messageIDs...).Err(); err != nil {
				log.Printf("[WORKER ERROR] failed to ack queue stream batch: %v", err)
			}
		}
	}
}
