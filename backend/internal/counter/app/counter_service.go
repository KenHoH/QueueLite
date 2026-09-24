package app

import (
	"QueueLite/internal/apperror"
	"QueueLite/internal/counter/domain"
	queueapp "QueueLite/internal/queue/app"
	queuedomain "QueueLite/internal/queue/domain"
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type CounterService struct {
	repo      CounterRepo
	queueRepo CounterQueueRepo
}

func NewCounterService(repo CounterRepo, queueRepo ...CounterQueueRepo) *CounterService {
	service := &CounterService{repo: repo}
	if len(queueRepo) > 0 {
		service.queueRepo = queueRepo[0]
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

	queue.CounterID = &counterID
	if queue.State == "" || queue.State == queuedomain.QueueStateWaiting {
		queue.State = queuedomain.QueueStateCalled
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

func (s *CounterService) DeleteCounter(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteCounter(ctx, id); err != nil {
		if errors.Is(err, ErrCounterNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "COUNTER_NOT_FOUND", "counter not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "DELETE_COUNTER_ERROR", "failed to delete counter", err)
	}
	return nil
}
