package app

import (
	"context"
	"errors"
	"strings"
	"time"

	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	"QueueLite/internal/counter/domain"
	queueapp "QueueLite/internal/queue/app"
	queuedomain "QueueLite/internal/queue/domain"
	subscriptionapp "QueueLite/internal/subscription/app"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CounterService struct {
	repo                CounterRepo
	queueRepo           CounterQueueRepo
	txManager           CounterTxManager
	notifier            CounterNotifier
	subscriptionService *subscriptionapp.SubscriptionService
	rdt                 *redis.Client
}

func NewCounterService(repo CounterRepo, queueRepo CounterQueueRepo, subscriptionService *subscriptionapp.SubscriptionService, redis *redis.Client) *CounterService {
	return &CounterService{
		repo:                repo,
		queueRepo:           queueRepo,
		subscriptionService: subscriptionService,
		rdt:                 redis,
	}
}

func (s *CounterService) SetNotifier(notifier CounterNotifier) {
	s.notifier = notifier
}

func (s *CounterService) SetTxManager(txManager CounterTxManager) {
	s.txManager = txManager
}

func (s *CounterService) runInTx(ctx context.Context, fn func(counterRepo CounterRepo, queueRepo CounterQueueRepo) error) error {
	if s.txManager != nil {
		return s.txManager.WithTransaction(ctx, fn)
	}
	return fn(s.repo, s.queueRepo)
}

func (s *CounterService) publishQueueUpdate(ctx context.Context, businessID uuid.UUID) {
	if s.rdt == nil {
		return
	}
	_ = cache.PublishMessage(ctx, s.rdt, cache.QueueEventChannel(businessID.String()), "queue.update")
}

func (s *CounterService) removeWaitingQueue(ctx context.Context, businessID uuid.UUID, queueID uuid.UUID) {
	if s.rdt == nil {
		return
	}
	_ = cache.RemoveWaitingQueue(ctx, s.rdt, businessID.String(), queueID.String())
}

func (s *CounterService) addWaitingQueue(ctx context.Context, queue queuedomain.Queue) {
	if s.rdt == nil {
		return
	}
	_ = cache.AddWaitingQueue(ctx, s.rdt, queue)
}

func (s *CounterService) getQueue(ctx context.Context, queueID uuid.UUID) (*queuedomain.Queue, error) {
	queue, err := s.queueRepo.GetQueue(ctx, queueID)
	if err != nil {
		if errors.Is(err, queueapp.ErrQueueNotFound) {
			return nil, apperror.Wrap(apperror.KindNotFound, "QUEUE_NOT_FOUND", "queue not found", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "GET_QUEUE_ERROR", "failed to get queue", err)
	}
	return queue, nil
}

func wrapCounterError(err error, code string, message string) error {
	if errors.Is(err, ErrCounterNotFound) {
		return apperror.Wrap(apperror.KindNotFound, "COUNTER_NOT_FOUND", "counter not found", err)
	}
	return apperror.Wrap(apperror.KindInternal, code, message, err)
}

func (s *CounterService) CreateCounter(ctx context.Context, counter domain.Counter) (*domain.Counter, error) {
	counter.Name = strings.TrimSpace(counter.Name)
	if counter.BusinessID == uuid.Nil {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
	}
	if counter.Name == "" {
		return nil, apperror.New(apperror.KindInvalid, "COUNTER_NAME_REQUIRED", "counter name is required")
	}

	record, err := s.repo.CreateCounter(ctx, &counter)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "CREATE_COUNTER_ERROR", "failed to create counter", err)
	}
	s.publishQueueUpdate(ctx, record.BusinessID)
	return record, nil
}

func (s *CounterService) GetCounter(ctx context.Context, id uuid.UUID) (*domain.Counter, error) {
	counter, err := s.repo.GetCounter(ctx, id)
	if err != nil {
		return nil, wrapCounterError(err, "GET_COUNTER_ERROR", "failed to get counter")
	}
	return counter, nil
}

func (s *CounterService) UpdateCounter(ctx context.Context, counter domain.Counter) error {
	counter.Name = strings.TrimSpace(counter.Name)
	if counter.Name == "" {
		return apperror.New(apperror.KindInvalid, "COUNTER_NAME_REQUIRED", "counter name is required")
	}
	if err := s.repo.UpdateCounter(ctx, &counter); err != nil {
		return wrapCounterError(err, "UPDATE_COUNTER_ERROR", "failed to update counter")
	}
	if counter.BusinessID != uuid.Nil {
		s.publishQueueUpdate(ctx, counter.BusinessID)
	}
	return nil
}

func (s *CounterService) UpdateCounterEmployee(ctx context.Context, counterID uuid.UUID, employeeID *uuid.UUID) error {
	if err := s.repo.UpdateCounterEmployee(ctx, counterID, employeeID); err != nil {
		return wrapCounterError(err, "UPDATE_COUNTER_EMPLOYEE_ERROR", "failed to update counter employee")
	}
	return nil
}

func (s *CounterService) ClearCounterCustomer(ctx context.Context, counterID uuid.UUID) error {
	if err := s.repo.UpdateCounterCustomer(ctx, counterID, nil); err != nil {
		return wrapCounterError(err, "UPDATE_COUNTER_CUSTOMER_ERROR", "failed to update counter customer")
	}
	return nil
}

func (s *CounterService) DeleteCounter(ctx context.Context, id uuid.UUID) error {
	existing, _ := s.repo.GetCounter(ctx, id)
	if err := s.repo.DeleteCounter(ctx, id); err != nil {
		return wrapCounterError(err, "DELETE_COUNTER_ERROR", "failed to delete counter")
	}
	if existing != nil && existing.BusinessID != uuid.Nil {
		s.publishQueueUpdate(ctx, existing.BusinessID)
	}
	return nil
}

func nowPtr() *time.Time {
	now := time.Now()
	return &now
}
