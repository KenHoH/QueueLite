package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	qr "QueueLite/internal/QRCode"
	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	businessdomain "QueueLite/internal/business/domain"

	"QueueLite/internal/queue/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type QueueService struct {
	repo                QueueRepo
	queueBusinessRepo   QueueBusinessRepo
	queueUserRepo       QueueUserRepo
	queueCounterRepo    QueueCounterRepo
	subscriptionService QueueQuota
	rdt                 *redis.Client
}

func NewQueueService(repo QueueRepo, subscriptionService QueueQuota, redis *redis.Client, queueBusinessRepo QueueBusinessRepo, queueUserRepo QueueUserRepo, queueCounterRepo QueueCounterRepo) *QueueService {
	service := &QueueService{
		repo:                repo,
		subscriptionService: subscriptionService,
		rdt:                 redis,
		queueBusinessRepo:   queueBusinessRepo,
		queueUserRepo:       queueUserRepo,
		queueCounterRepo:    queueCounterRepo,
	}
	return service
}

func (s *QueueService) GenerateQRImage(url string, businessID string) ([]byte, error) {
	fullURL := fmt.Sprintf("%s/queuelite/qr/%s", url, businessID)
	var file []byte
	file, err := qr.GenerateQRCode(fullURL)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (s *QueueService) GetCustomerQueueStatus(ctx context.Context, queueID uuid.UUID) (*CustomerQueueStatus, error) {
	data, err := s.loadCustomerQueueStatusData(ctx, queueID)
	if err != nil {
		return nil, err
	}
	queuesByID, activeQueues := indexQueues(data.queues)
	counterStatuses, serving, activeCounters := buildCounterStatus(data.counters, queuesByID)

	var next *QueueSnapshotItem

	if len(data.waiting) > 0 {
		value := data.waiting[0]
		next = &value
	}

	var position *CustomerQueuePosition
	var estimate *int

	if data.queue.State == domain.QueueStateWaiting {
		for index, item := range data.waiting {
			if item.QueueID == data.queue.ID.String() {
				position = &CustomerQueuePosition{
					Position:  index + 1,
					Ahead:     index,
					QueueName: data.queue.Name,
				}

				activeCounterCount := max(activeCounters, 1)
				minutes := ((index + activeCounterCount - 1) / activeCounterCount) * 5
				estimate = &minutes
				break
			}
		}
	}

	return &CustomerQueueStatus{
		Queue:                *data.queue,
		Business:             *data.business,
		Counters:             counterStatuses,
		TotalCounters:        len(counterStatuses),
		ActiveCounters:       activeCounters,
		CurrentlyServing:     serving,
		NextQueue:            next,
		CustomerPosition:     position,
		TotalWaiting:         len(data.waiting),
		TotalActiveQueues:    len(data.waiting) + activeQueues,
		EstimatedWaitMinutes: estimate,
		UpdatedAt:            time.Now().UTC(),
	}, nil
}

func (s *QueueService) ValidateBusinessExists(ctx context.Context, businessID uuid.UUID) error {
	if businessID == uuid.Nil {
		return apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
	}
	if s.queueBusinessRepo == nil {
		return apperror.New(apperror.KindInternal, "BUSINESS_LOOKUP_UNAVAILABLE", "business lookup unavailable")
	}
	if _, err := s.queueBusinessRepo.GetBusiness(ctx, businessID); err != nil {
		if errors.Is(err, businessdomain.ErrBusinessNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "BUSINESS_NOT_FOUND", "business not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_ERROR", "failed to get business", err)
	}
	return nil
}

// UserID is supplied only by the authenticated HTTP context, never decoded from JSON.
func (s *QueueService) RegisterCustomerQueue(ctx context.Context, input RegisterCustomerQueueInput) (*RegisterCustomerQueueResult, error) {
	business, err := s.validateCustomerQueueBusiness(ctx, input.BusinessID)
	if err != nil {
		return nil, err
	}

	customer, err := s.resolveQueueCustomer(ctx, input)
	if err != nil {
		return nil, err
	}

	queue, err := s.prepareCustomerQueue(
		ctx,
		business.ID,
		customer,
	)
	if err != nil {
		return nil, err
	}

	if err := s.registerCustomerQueueAtomically(ctx, queue, customer); err != nil {
		return nil, err
	}

	// Registration has succeeded. Notifications are best-effort.
	_ = s.publishQueueUpdate(ctx, queue.BusinessID)

	return &RegisterCustomerQueueResult{
		Queue:       &queue,
		GuestID:     customer.GuestID,
		Username:    customer.Username,
		PhoneNumber: customer.PhoneNumber,
	}, nil
}

func (s *QueueService) RegisterQueue(ctx context.Context, queue domain.Queue) (*domain.Queue, error) {
	if queue.BusinessID == uuid.Nil {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
	}
	if queue.UserID == nil {
		return nil, apperror.New(apperror.KindInvalid, "QUEUE_OWNER_REQUIRED", "user id is required")
	}

	// check if user have joined the queue on that business
	active, err := s.repo.GetActiveQueueByUserAndBusiness(ctx, *queue.UserID, queue.BusinessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_ACTIVE_QUEUE_ERROR", "failed to check active queue", err)
	}
	if active != nil {
		return nil, apperror.New(apperror.KindInvalid, "ACTIVE_QUEUE_EXISTS", "user already has active queue in this business")
	}

	if s.subscriptionService != nil {
		// check if the businessQuota is valid or not
		if err := s.subscriptionService.CheckBusinessQueueQuota(ctx, queue.BusinessID); err != nil {
			return nil, apperror.New(apperror.KindInvalid, "BUSINESS_QUEUE_FULL", "the business queueu is full")
		}

		// if the user choose priority
		if queue.Priority {
			if err := s.subscriptionService.CheckUserPrioritySlot(ctx, *queue.UserID); err != nil {
				return nil, apperror.New(apperror.KindInvalid, "INVALID_PRIORITY_QUOTA", "Insufficient priority slot")
			}
		}
	}

	// set default state for the queue
	if queue.State == "" {
		queue.State = domain.QueueStateWaiting
	}
	if queue.ID == uuid.Nil {
		queue.ID = uuid.New()
	}
	if queue.CreatedAt.IsZero() {
		queue.CreatedAt = time.Now()
	}

	// generate the queue name
	if strings.TrimSpace(queue.Name) == "" {
		floor, err := s.repo.GetDailyQueueNumberFloor(ctx, queue.BusinessID, queue.CreatedAt)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "GENERATE_QUEUE_NAME_ERROR", "failed to allocate queue number", err)
		}
		name, err := cache.GenerateDailyQueueName(ctx, s.rdt, queue.BusinessID.String(), queue.CreatedAt, floor)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "GENERATE_QUEUE_NAME_ERROR", "failed to generate queue name", err)
		}
		queue.Name = name
	}

	quotaDecreased := false
	if s.subscriptionService != nil {
		if err := s.subscriptionService.DecreaseBusinessCapacity(ctx, queue.BusinessID); err != nil {
			return nil, err
		}
		quotaDecreased = true
	}
	rollbackQuota := func() {
		if quotaDecreased && s.subscriptionService != nil {
			_ = s.subscriptionService.IncreaseBusinessCapacity(ctx, queue.BusinessID)
		}
	}

	// insert this queue to the redis stream to write to db
	if err := cache.AddQueueStream(ctx, s.rdt, queue); err != nil {
		rollbackQuota()
		return nil, apperror.Wrap(apperror.KindInternal, "REGISTER_QUEUE_STREAM_ERROR", "failed to queue registration for database worker", err)
	}

	// adding to the queue cache layer
	if err := s.syncQueueCache(ctx, queue); err != nil {
		rollbackQuota()
		return nil, apperror.Wrap(apperror.KindInternal, "CACHE_QUEUE_ERROR", "failed to register queue to redis", err)
	}

	// send messages to trigger the sub channel
	if err := s.publishQueueUpdate(ctx, queue.BusinessID); err != nil {
		rollbackQuota()
		return nil, apperror.Wrap(apperror.KindInternal, "PUBLISH_MESSAGE_ERROR", "failed to publish queue", err)
	}
	return &queue, nil
}

func (s *QueueService) GetQueue(ctx context.Context, id uuid.UUID) (*domain.Queue, error) {
	queue, err := s.repo.GetQueue(ctx, id)
	if err != nil {
		if errors.Is(err, ErrQueueNotFound) {
			if s.rdt != nil {
				customer, cacheErr := cache.GetCustomerQueue(ctx, s.rdt, id.String())
				if cacheErr == nil {
					return &customer.Queue, nil
				}
				if !errors.Is(cacheErr, redis.Nil) {
					return nil, apperror.Wrap(apperror.KindInternal, "GET_QUEUE_CACHE_ERROR", "failed to get queue", cacheErr)
				}
			}
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

func (s *QueueService) GetActiveQueuesByUser(ctx context.Context, userID uuid.UUID) ([]domain.Queue, error) {
	queues, err := s.repo.GetActiveQueuesByUser(ctx, userID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_USER_QUEUES_ERROR", "failed to get your queues", err)
	}
	return queues, nil
}

func (s *QueueService) UpdateState(ctx context.Context, queueID uuid.UUID, state domain.QueueState) error {
	if err := s.requirePersistedQueue(ctx, queueID); err != nil {
		return err
	}
	queue, err := s.GetQueue(ctx, queueID)
	if err != nil {
		return err
	}

	queue.State = state
	queue.StampLifecycle(time.Now())

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
	if err := s.requirePersistedQueue(ctx, queueID); err != nil {
		return err
	}
	queue, err := s.GetQueue(ctx, queueID)
	if err != nil {
		return err
	}
	queue.State = domain.QueueStateCancelled
	queue.StampLifecycle(time.Now())
	if err := s.repo.UpdateQueue(ctx, queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "CANCEL_QUEUE_ERROR", "failed to cancel queue", err)
	}
	if err := s.syncQueueCache(ctx, *queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "CANCEL_QUEUE_CACHE_ERROR", "failed to update queue cache", err)
	}
	_ = s.publishQueueUpdate(ctx, queue.BusinessID)
	return nil
}

// Customer cancellation and Call Next compare the persisted waiting state in
// the same database update. Only one transition can win that race.
func (s *QueueService) CancelWaitingQueue(ctx context.Context, queueID uuid.UUID) error {
	if err := s.requirePersistedQueue(ctx, queueID); err != nil {
		return err
	}
	queue, err := s.GetQueue(ctx, queueID)
	if err != nil {
		return err
	}
	if queue.State != domain.QueueStateCancelled {
		if queue.State != domain.QueueStateWaiting {
			return apperror.New(apperror.KindConflict, "QUEUE_NOT_CANCELLABLE", "queue is no longer waiting")
		}
		queue.State = domain.QueueStateCancelled
		queue.StampLifecycle(time.Now())
		if err := s.repo.UpdateQueueIfState(ctx, queue, domain.QueueStateWaiting); err != nil {
			return err
		}
	}
	if err := s.syncQueueCache(ctx, *queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "CANCEL_QUEUE_CACHE_ERROR", "failed to release queue reservation", err)
	}
	_ = s.publishQueueUpdate(ctx, queue.BusinessID)
	return nil
}

func (s *QueueService) UpdateQueueIfState(ctx context.Context, queue *domain.Queue, expected domain.QueueState) error {
	return s.repo.UpdateQueueIfState(ctx, queue, domain.QueueStateWaiting)
}

func (s *QueueService) MarkAsSkipped(ctx context.Context, queueID uuid.UUID) error {
	return s.UpdateState(ctx, queueID, domain.QueueStateSkipped)
}

func (s *QueueService) UpdateQueue(ctx context.Context, queue domain.Queue) error {
	if err := s.requirePersistedQueue(ctx, queue.ID); err != nil {
		return err
	}
	if queue.Name == "" {
		return apperror.New(apperror.KindInvalid, "QUEUE_NAME_REQUIRED", "missing queue name")
	}
	if queue.State == "" {
		queue.State = domain.QueueStateWaiting
	}
	queue.StampLifecycle(time.Now())
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
	if err := s.requirePersistedQueue(ctx, id); err != nil {
		return err
	}
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
	if err := cache.ReleaseCustomerQueue(ctx, s.rdt, *queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "RELEASE_QUEUE_ERROR", "failed to release queue reservation", err)
	}
	if err := s.rdt.Del(ctx, cache.CustomerQueueKey(id.String()), cache.QueueNameKey(id.String())).Err(); err != nil {
		return apperror.Wrap(apperror.KindInternal, "DELETE_QUEUE_CACHE_ERROR", "failed to remove queue cache", err)
	}
	_ = cache.RemoveWaitingQueue(ctx, s.rdt, queue.BusinessID.String(), queue.ID.String())
	_ = s.publishQueueUpdate(ctx, queue.BusinessID)
	return nil
}
