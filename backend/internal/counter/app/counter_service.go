package app

import (
	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	"QueueLite/internal/counter/domain"
	queueapp "QueueLite/internal/queue/app"
	queuedomain "QueueLite/internal/queue/domain"
	subscriptionapp "QueueLite/internal/subscription/app"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CounterService struct {
	repo                CounterRepo
	queueRepo           CounterQueueRepo
	subscriptionService *subscriptionapp.SubscriptionService
	rdt                 *redis.Client
}

func NewCounterService(repo CounterRepo, queueRepo CounterQueueRepo, subscriptionService *subscriptionapp.SubscriptionService, redis *redis.Client) *CounterService {
	service := &CounterService{
		repo:                repo,
		queueRepo:           queueRepo,
		subscriptionService: subscriptionService,
		rdt:                 redis,
	}
	return service
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
	return record, nil
}

func (s *CounterService) GetCounter(ctx context.Context, id uuid.UUID) (*domain.Counter, error) {
	counter, err := s.repo.GetCounter(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCounterNotFound) {
			return nil, apperror.Wrap(apperror.KindNotFound, "COUNTER_NOT_FOUND", "counter not found", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "GET_COUNTER_ERROR", "failed to get counter", err)
	}
	return counter, nil
}

func (s *CounterService) UpdateCounter(ctx context.Context, counter domain.Counter) error {
	counter.Name = strings.TrimSpace(counter.Name)
	if counter.Name == "" {
		return apperror.New(apperror.KindInvalid, "COUNTER_NAME_REQUIRED", "counter name is required")
	}
	if err := s.repo.UpdateCounter(ctx, &counter); err != nil {
		if errors.Is(err, ErrCounterNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "COUNTER_NOT_FOUND", "counter not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "UPDATE_COUNTER_ERROR", "failed to update counter", err)
	}
	return nil
}

func (s *CounterService) UpdateCounterEmployee(ctx context.Context, counterID uuid.UUID, employeeID *uuid.UUID) error {
	if err := s.repo.UpdateCounterEmployee(ctx, counterID, employeeID); err != nil {
		if errors.Is(err, ErrCounterNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "COUNTER_NOT_FOUND", "counter not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "UPDATE_COUNTER_EMPLOYEE_ERROR", "failed to update counter employee", err)
	}
	return nil
}

func (s *CounterService) UpdateCounterCustomer(ctx context.Context, counterID uuid.UUID, queueID uuid.UUID) error {
	counter, err := s.GetCounter(ctx, counterID)
	if err != nil {
		return err
	}

	queue, err := s.queueRepo.GetQueue(ctx, queueID)
	if err != nil {
		if errors.Is(err, queueapp.ErrQueueNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "QUEUE_NOT_FOUND", "queue not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "GET_QUEUE_ERROR", "failed to get queue", err)
	}

	if queue.BusinessID != counter.BusinessID {
		return apperror.New(apperror.KindInvalid, "COUNTER_QUEUE_BUSINESS_MISMATCH", "counter and queue belong to different businesses")
	}

	wasWaiting := queue.State == "" || queue.State == queuedomain.QueueStateWaiting
	if wasWaiting {
		queue.State = queuedomain.QueueStateCalled
		queue.CalledByCounterID = &counterID
	}

	if err := s.queueRepo.UpdateQueue(ctx, queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "UPDATE_COUNTER_CUSTOMER_QUEUE_ERROR", "failed to assign queue to counter", err)
	}

	if err := s.repo.UpdateCounterCustomer(ctx, counterID, &queueID); err != nil {
		if errors.Is(err, ErrCounterNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "COUNTER_NOT_FOUND", "counter not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "UPDATE_COUNTER_CUSTOMER_ERROR", "failed to update counter customer", err)
	}
	if wasWaiting {
		s.removeWaitingQueue(ctx, queue.BusinessID, queue.ID)
		s.publishQueueUpdate(ctx, queue.BusinessID)
	}
	return nil
}

func (s *CounterService) ClearCounterCustomer(ctx context.Context, counterID uuid.UUID) error {
	if err := s.repo.UpdateCounterCustomer(ctx, counterID, nil); err != nil {
		if errors.Is(err, ErrCounterNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "COUNTER_NOT_FOUND", "counter not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "UPDATE_COUNTER_CUSTOMER_ERROR", "failed to update counter customer", err)
	}
	return nil
}

func (s *CounterService) RemoveQueueFromCounter(ctx context.Context, counterID uuid.UUID, queueID uuid.UUID) error {
	if s.queueRepo == nil {
		return apperror.New(apperror.KindNotImplemented, "QUEUE_REPO_NOT_CONFIGURED", "queue repo is not configured")
	}

	counter, err := s.GetCounter(ctx, counterID)
	if err != nil {
		return err
	}

	queue, err := s.queueRepo.GetQueue(ctx, queueID)
	if err != nil {
		if errors.Is(err, queueapp.ErrQueueNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "QUEUE_NOT_FOUND", "queue not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "GET_QUEUE_ERROR", "failed to get queue", err)
	}

	if queue.CalledByCounterID == nil || *queue.CalledByCounterID != counterID {
		return apperror.New(apperror.KindInvalid, "QUEUE_COUNTER_MISMATCH", "queue was not called by this counter")
	}

	queue.CalledByCounterID = nil
	queue.State = queuedomain.QueueStateCompleted
	queue.DoneAt = nowPtr()
	if err := s.queueRepo.UpdateQueue(ctx, queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "REMOVE_QUEUE_COUNTER_ERROR", "failed to remove queue from counter", err)
	}

	s.removeWaitingQueue(ctx, queue.BusinessID, queue.ID)
	s.publishQueueUpdate(ctx, queue.BusinessID)
	if counter.CurrentQueueID != nil && *counter.CurrentQueueID == queueID {
		return s.ClearCounterCustomer(ctx, counterID)
	}
	return nil
}

func (s *CounterService) CallNextQueue(ctx context.Context, counterID uuid.UUID, businessID uuid.UUID) (*queuedomain.Queue, error) {
	if s.queueRepo == nil {
		return nil, apperror.New(apperror.KindNotImplemented, "QUEUE_REPO_NOT_CONFIGURED", "queue repo is not configured")
	}
	counter, err := s.GetCounter(ctx, counterID)
	if err != nil {
		return nil, err
	}
	if counter.BusinessID != businessID {
		return nil, apperror.New(apperror.KindInvalid, "COUNTER_BUSINESS_MISMATCH", "counter does not belong to business")
	}
	if counter.CurrentQueueID != nil {
		current, err := s.queueRepo.GetQueue(ctx, *counter.CurrentQueueID)
		if err == nil {
			current.State = queuedomain.QueueStateCompleted
			current.DoneAt = nowPtr()
			_ = s.queueRepo.UpdateQueue(ctx, current)
		}
		if err := s.repo.UpdateCounterCustomer(ctx, counterID, nil); err != nil {
			return nil, err
		}
	}

	queueItem, err := cache.PopNextFairWaitingQueueWithScore(ctx, s.rdt, businessID.String())
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "POP_TOP_QUEUE_ERROR", "failed to pop top waiting queue", err)
	}
	if queueItem == nil {
		return nil, nil
	}
	queueIDString, ok := queueItem.Member.(string)
	if !ok {
		_ = cache.RestoreWaitingQueueItem(ctx, s.rdt, *queueItem)
		return nil, apperror.New(apperror.KindInternal, "INVALID_QUEUE_CACHE_ID", "invalid queue id in waiting cache")
	}
	queueID, err := uuid.Parse(queueIDString)
	if err != nil {
		_ = cache.RestoreWaitingQueueItem(ctx, s.rdt, *queueItem)
		return nil, apperror.Wrap(apperror.KindInternal, "INVALID_QUEUE_CACHE_ID", "invalid queue id in waiting cache", err)
	}

	next, err := s.getQueueWithRetry(ctx, queueID)
	if err != nil {
		_ = cache.RestoreWaitingQueueItem(ctx, s.rdt, *queueItem)
		return nil, err
	}
	next.State = queuedomain.QueueStateCalled
	next.CalledByCounterID = &counterID
	next.CalledAt = nowPtr()
	if err := s.queueRepo.UpdateQueue(ctx, next); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "CALL_NEXT_QUEUE_ERROR", "failed to call next queue", err)
	}
	if err := s.repo.UpdateCounterCustomer(ctx, counterID, &next.ID); err != nil {
		return nil, err
	}
	s.publishQueueUpdate(ctx, next.BusinessID)
	s.scheduleCalledQueueTimeout(next.ID, counterID)
	return next, nil
}

func (s *CounterService) ProcessCalledQueue(ctx context.Context, counterID uuid.UUID, queueID uuid.UUID) (*queuedomain.Queue, error) {
	if s.queueRepo == nil {
		return nil, apperror.New(apperror.KindNotImplemented, "QUEUE_REPO_NOT_CONFIGURED", "queue repo is not configured")
	}
	queue, err := s.queueRepo.GetQueue(ctx, queueID)
	if err != nil {
		return nil, err
	}
	if queue.State != queuedomain.QueueStateCalled {
		return nil, apperror.New(apperror.KindInvalid, "QUEUE_NOT_CALLED", "queue is not called")
	}
	if queue.CalledByCounterID == nil || *queue.CalledByCounterID != counterID {
		return nil, apperror.New(apperror.KindInvalid, "QUEUE_COUNTER_MISMATCH", "queue was not called by this counter")
	}
	queue.State = queuedomain.QueueStateProcessing
	queue.ProcessingAt = nowPtr()
	if err := s.queueRepo.UpdateQueue(ctx, queue); err != nil {
		return nil, err
	}
	if s.subscriptionService != nil && queue.Priority && queue.UserID != nil {
		if err := s.subscriptionService.DecreaseUserSlot(ctx, *queue.UserID); err != nil {
			return nil, err
		}
	}
	if err := s.repo.UpdateCounterCustomer(ctx, counterID, &queueID); err != nil {
		return nil, err
	}
	s.removeWaitingQueue(ctx, queue.BusinessID, queue.ID)
	return queue, nil
}

func (s *CounterService) SkipQueue(ctx context.Context, counterID uuid.UUID, queueID uuid.UUID) (*queuedomain.Queue, error) {
	if s.queueRepo == nil {
		return nil, apperror.New(apperror.KindNotImplemented, "QUEUE_REPO_NOT_CONFIGURED", "queue repo is not configured")
	}
	queue, err := s.queueRepo.GetQueue(ctx, queueID)
	if err != nil {
		return nil, err
	}
	if queue.CalledByCounterID == nil || *queue.CalledByCounterID != counterID {
		return nil, apperror.New(apperror.KindInvalid, "QUEUE_COUNTER_MISMATCH", "queue was not called by this counter")
	}
	if queue.State != queuedomain.QueueStateCalled && queue.State != queuedomain.QueueStateProcessing {
		return nil, apperror.New(apperror.KindInvalid, "QUEUE_INVALID_STATE", "queue cannot be skipped")
	}
	queue.State = queuedomain.QueueStateSkipped
	queue.CancelledAt = nowPtr()
	if err := s.queueRepo.UpdateQueue(ctx, queue); err != nil {
		return nil, err
	}
	s.removeWaitingQueue(ctx, queue.BusinessID, queue.ID)
	counter, err := s.GetCounter(ctx, counterID)
	if err == nil && counter.CurrentQueueID != nil && *counter.CurrentQueueID == queueID {
		_ = s.repo.UpdateCounterCustomer(ctx, counterID, nil)
	}
	next, err := s.CallNextQueue(ctx, counterID, queue.BusinessID)
	if err != nil {
		return nil, err
	}
	if next == nil {
		s.publishQueueUpdate(ctx, queue.BusinessID)
	}
	return next, nil
}

func (s *CounterService) getQueueWithRetry(ctx context.Context, queueID uuid.UUID) (*queuedomain.Queue, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		queue, err := s.queueRepo.GetQueue(ctx, queueID)
		if err == nil {
			return queue, nil
		}
		if !errors.Is(err, queueapp.ErrQueueNotFound) {
			return nil, err
		}
		lastErr = err
		if attempt < 2 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(3 * time.Second):
			}
		}
	}
	return nil, apperror.Wrap(apperror.KindNotFound, "QUEUE_NOT_READY", "queue is not ready in database yet", lastErr)
}

func (s *CounterService) scheduleCalledQueueTimeout(queueID uuid.UUID, counterID uuid.UUID) {
	time.AfterFunc(5*time.Minute, func() {
		ctx := context.Background()
		if s.queueRepo == nil {
			return
		}
		queue, err := s.queueRepo.GetQueue(ctx, queueID)
		if err != nil || queue.State != queuedomain.QueueStateCalled {
			return
		}
		_, _ = s.SkipQueue(ctx, counterID, queueID)
	})
}

func nowPtr() *time.Time {
	now := time.Now()
	return &now
}

func (s *CounterService) DeleteCounter(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteCounter(ctx, id); err != nil {
		if errors.Is(err, ErrCounterNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "COUNTER_NOT_FOUND", "counter not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "DELETE_COUNTER_ERROR", "failed to delete counter", err)
	}
	return nil
}
