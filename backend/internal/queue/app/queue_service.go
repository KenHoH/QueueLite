package app

import (
	"QueueLite/internal/apperror"
	"QueueLite/internal/queue/domain"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type QueueService struct {
	repo QueueRepo
}

func NewQueueService(repo QueueRepo) *QueueService {
	return &QueueService{
		repo: repo,
	}
}

func (s *QueueService) RegisterQueue(ctx context.Context, queue domain.Queue) (*domain.Queue, error) {
	if queue.State == "" {
		queue.State = domain.QueueStateWaiting
	}

	record, err := s.repo.CreateQueue(ctx, &queue)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "REGISTER_QUEUE_ERROR", "failed to register queue", err)
	}

	return record, nil
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

func (s *QueueService) UpdateState(ctx context.Context, queueID uuid.UUID, state domain.QueueState) error {
	queue, err := s.GetQueue(ctx, queueID)
	if err != nil {
		return err
	}

	now := time.Now()
	queue.State = state
	if state == domain.QueueStateProcess && queue.StartTime == nil {
		queue.StartTime = &now
	}
	if state == domain.QueueStateCompleted || state == domain.QueueStateCanceled {
		queue.EndTime = &now
	}

	if err := s.repo.UpdateQueue(ctx, queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "UPDATE_QUEUE_STATE_ERROR", "failed to update queue state", err)
	}

	return nil
}

func (s *QueueService) MarkAsDone(ctx context.Context, queueID uuid.UUID) error {
	return s.UpdateState(ctx, queueID, domain.QueueStateCompleted)
}

func (s *QueueService) AssignToCounter(ctx context.Context, queueID uuid.UUID, counterID uuid.UUID) error {
	queue, err := s.GetQueue(ctx, queueID)
	if err != nil {
		return err
	}

	queue.CounterID = &counterID
	if queue.State == "" || queue.State == domain.QueueStateWaiting {
		queue.State = domain.QueueStateCalled
	}

	if err := s.repo.UpdateQueue(ctx, queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "ASSIGN_QUEUE_COUNTER_ERROR", "failed to assign queue to counter", err)
	}

	return nil
}

func (s *QueueService) RemoveFromCounter(ctx context.Context, queueID uuid.UUID, counterID uuid.UUID) error {
	queue, err := s.GetQueue(ctx, queueID)
	if err != nil {
		return err
	}

	if queue.CounterID == nil || *queue.CounterID != counterID {
		return apperror.New(apperror.KindInvalid, "QUEUE_COUNTER_MISMATCH", "queue is not assigned to counter")
	}

	queue.CounterID = nil
	if queue.State == domain.QueueStateCalled {
		queue.State = domain.QueueStateWaiting
	}

	if err := s.repo.UpdateQueue(ctx, queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "REMOVE_QUEUE_COUNTER_ERROR", "failed to remove queue from counter", err)
	}

	return nil
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
