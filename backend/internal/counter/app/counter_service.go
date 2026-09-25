package app

import (
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
)

type CounterService struct {
	repo                CounterRepo
	queueRepo           CounterQueueRepo
	subscriptionService *subscriptionapp.SubscriptionService
}

func NewCounterService(repo CounterRepo, queueRepo CounterQueueRepo, subscriptionService ...*subscriptionapp.SubscriptionService) *CounterService {
	service := &CounterService{repo: repo, queueRepo: queueRepo}
	if len(subscriptionService) > 0 {
		service.subscriptionService = subscriptionService[0]
	}
	return service
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

	if queue.BusinessID != counter.BusinessID {
		return apperror.New(apperror.KindInvalid, "COUNTER_QUEUE_BUSINESS_MISMATCH", "counter and queue belong to different businesses")
	}

	if queue.State == "" || queue.State == queuedomain.QueueStateWaiting {
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
			now := nowPtr()
			current.DoneAt = now
			_ = s.queueRepo.UpdateQueue(ctx, current)
		}
		if err := s.repo.UpdateCounterCustomer(ctx, counterID, nil); err != nil {
			return nil, err
		}
	}
	next, err := s.queueRepo.GetTopQueueByBusinessPrivateForUpdate(ctx, businessID)
	if err != nil || next == nil {
		return next, err
	}
	next.State = queuedomain.QueueStateCalled
	next.CalledByCounterID = &counterID
	next.CalledAt = nowPtr()
	if err := s.queueRepo.UpdateQueue(ctx, next); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "CALL_NEXT_QUEUE_ERROR", "failed to call next queue", err)
	}
	_ = PublishQueueEvent(ctx, "queue.called", next)
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
	if s.subscriptionService != nil {
		if err := s.subscriptionService.DecreaseBusinessCapacity(ctx, queue.BusinessID); err != nil {
			return nil, err
		}
		if queue.Priority && queue.UserID != nil {
			if err := s.subscriptionService.DecreaseUserSlot(ctx, *queue.UserID); err != nil {
				return nil, err
			}
		}
	}
	if err := s.repo.UpdateCounterCustomer(ctx, counterID, &queueID); err != nil {
		return nil, err
	}
	_ = PublishQueueEvent(ctx, "queue.processing", queue)
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
	counter, err := s.GetCounter(ctx, counterID)
	if err == nil && counter.CurrentQueueID != nil && *counter.CurrentQueueID == queueID {
		_ = s.repo.UpdateCounterCustomer(ctx, counterID, nil)
	}
	_ = PublishQueueEvent(ctx, "queue.skipped", queue)
	return s.CallNextQueue(ctx, counterID, queue.BusinessID)
}

// TODO:implement
func PublishQueueEvent(ctx context.Context, eventName string, payload any) error {
	return nil
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
