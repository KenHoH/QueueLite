package app

import (
	"context"
	"errors"
	"strings"
	"time"

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

// helper function cache redis related
func (s *QueueService) QueueExistsForAccess(ctx context.Context, queueID uuid.UUID) (bool, error) {
	existsInCache, err := cache.QueueExists(ctx, s.rdt, queueID.String())
	if err == nil && existsInCache {
		return true, nil
	}

	_, err = s.repo.GetQueue(ctx, queueID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrQueueNotFound) {
		return false, nil
	}
	return false, err
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

func (s *QueueService) GetCustomerQueueStatus(ctx context.Context, queueID uuid.UUID) (*CustomerQueueStatus, error) {
	queue, err := s.GetQueue(ctx, queueID)
	if err != nil {
		return nil, err
	}
	business, err := s.queueBusinessRepo.GetBusiness(ctx, queue.BusinessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_ERROR", "failed to get business", err)
	}
	counters, err := s.queueCounterRepo.ListBusinessCounters(ctx, queue.BusinessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_COUNTERS_ERROR", "failed to get counters", err)
	}
	queues, err := s.repo.GetAllQueueByBusiness(ctx, queue.BusinessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_QUEUES_ERROR", "failed to get business queues", err)
	}
	waiting, err := s.GetWaitingQueueSnapshot(ctx, queue.BusinessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_WAITING_QUEUE_ERROR", "failed to get waiting queues", err)
	}

	byID := make(map[uuid.UUID]domain.Queue, len(queues))
	calledOrProcessing := 0
	for _, item := range queues {
		byID[item.ID] = item
		if item.State == domain.QueueStateCalled || item.State == domain.QueueStateProcessing {
			calledOrProcessing++
		}
	}
	counterStatuses := make([]CustomerCounterStatus, 0, len(counters))
	serving := make([]CurrentlyServingQueue, 0, len(counters))
	activeCounters := 0
	for _, counter := range counters {
		item := CustomerCounterStatus{ID: counter.ID, Name: counter.Name, CurrentQueueID: counter.CurrentQueueID, State: "idle"}
		if counter.CurrentQueueID != nil {
			if current, ok := byID[*counter.CurrentQueueID]; ok && (current.State == domain.QueueStateCalled || current.State == domain.QueueStateProcessing) {
				name := current.Name
				item.CurrentQueueName = &name
				item.State = string(current.State)
				activeCounters++
				serving = append(serving, CurrentlyServingQueue{CounterID: counter.ID, CounterName: counter.Name, QueueID: current.ID, QueueName: current.Name, State: current.State})
			}
		}
		counterStatuses = append(counterStatuses, item)
	}

	var next *QueueSnapshotItem
	if len(waiting) > 0 {
		value := waiting[0]
		next = &value
	}
	var position *CustomerQueuePosition
	var estimate *int
	if queue.State == domain.QueueStateWaiting {
		for index, item := range waiting {
			if item.QueueID == queue.ID.String() {
				position = &CustomerQueuePosition{Position: index + 1, Ahead: index, QueueName: queue.Name}
				minutes := ((index + max(activeCounters, 1) - 1) / max(activeCounters, 1)) * 5
				estimate = &minutes
				break
			}
		}
	}
	return &CustomerQueueStatus{
		Queue: *queue, Business: *business, Counters: counterStatuses,
		TotalCounters: len(counterStatuses), ActiveCounters: activeCounters,
		CurrentlyServing: serving, NextQueue: next, CustomerPosition: position,
		TotalWaiting: len(waiting), TotalActiveQueues: len(waiting) + calledOrProcessing,
		EstimatedWaitMinutes: estimate, UpdatedAt: time.Now().UTC(),
	}, nil
}

func (s *QueueService) GetQueueCache(ctx context.Context, businessID string) ([]redis.Z, error) {
	return cache.GetWaitingQueues(ctx, businessID, s.rdt)
}

func (s *QueueService) GetQueueNameCache(ctx context.Context, queueID string) (string, error) {
	return cache.GetQueueName(ctx, s.rdt, queueID)
}

func (s *QueueService) SubscribeQueueChannel(ctx context.Context, businessID string) *redis.PubSub {
	return cache.SubscribeChannel(ctx, s.rdt, cache.QueueEventChannel(businessID))
}

func (s *QueueService) publishQueueUpdate(ctx context.Context, businessID uuid.UUID) error {
	return cache.PublishMessage(ctx, s.rdt, cache.QueueEventChannel(businessID.String()), "queue.update")
}

func (s *QueueService) syncQueueCache(ctx context.Context, queue domain.Queue) error {
	if queue.State == domain.QueueStateWaiting {
		return cache.AddWaitingQueue(ctx, s.rdt, queue) // adding to queue cache
	}
	if err := cache.SyncCustomerQueue(ctx, s.rdt, queue); err != nil {
		return err
	}
	return cache.RemoveWaitingQueue(ctx, s.rdt, queue.BusinessID.String(), queue.ID.String())
} // end of helper function

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
	if err := s.ValidateBusinessExists(ctx, input.BusinessID); err != nil {
		return nil, err
	}
	business, err := s.queueBusinessRepo.GetBusiness(ctx, input.BusinessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_ERROR", "failed to get business", err)
	}
	if !business.Operational {
		return nil, apperror.New(apperror.KindConflict, "BUSINESS_UNAVAILABLE", "business is unavailable")
	}
	ownerID := uuid.New()
	var guestID *uuid.UUID
	username, phone := strings.TrimSpace(input.Username), input.PhoneNumber
	if input.UserID != nil {
		if s.queueUserRepo == nil {
			return nil, apperror.New(apperror.KindInternal, "USER_LOOKUP_UNAVAILABLE", "user lookup unavailable")
		}
		user, err := s.queueUserRepo.GetUser(ctx, *input.UserID)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindUnauthorized, "USER_NOT_FOUND", "user not found", err)
		}
		ownerID, username, phone = user.ID, user.Username, user.PhoneNumber
	} else {
		guestID = &ownerID
	}
	if strings.TrimSpace(username) == "" {
		return nil, apperror.New(apperror.KindInvalid, "USERNAME_REQUIRED", "username is required")
	}
	if strings.TrimSpace(phone) == "" {
		return nil, apperror.New(apperror.KindInvalid, "PHONE_NUMBER_REQUIRED", "phone number is required")
	}
	normalized, err := NormalizeIndonesiaPhone(phone)
	if err != nil {
		return nil, err
	}
	active, err := s.repo.GetActiveQueueByUserAndBusiness(ctx, ownerID, input.BusinessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_ACTIVE_QUEUE_ERROR", "failed to check active queue", err)
	}
	if active != nil {
		return nil, apperror.New(apperror.KindConflict, "ACTIVE_QUEUE_EXISTS", "user already has active queue in this business")
	}
	q := domain.Queue{ID: uuid.New(), BusinessID: input.BusinessID, UserID: &ownerID, State: domain.QueueStateWaiting, Priority: false, CreatedAt: time.Now().UTC()}
	floor, err := s.repo.GetDailyQueueNumberFloor(ctx, q.BusinessID, q.CreatedAt)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GENERATE_QUEUE_NAME_ERROR", "failed to allocate queue number", err)
	}
	q.Name, err = cache.GenerateDailyQueueName(ctx, s.rdt, q.BusinessID.String(), q.CreatedAt, floor)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GENERATE_QUEUE_NAME_ERROR", "failed to allocate queue number", err)
	}
	if s.subscriptionService != nil {
		if err := s.subscriptionService.DecreaseBusinessCapacity(ctx, q.BusinessID); err != nil {
			return nil, customerQuotaError(err)
		}
	}
	outcome, err := cache.EnqueueCustomerQueue(ctx, s.rdt, cache.CustomerQueue{Queue: q, Username: username, PhoneNumber: normalized, Guest: guestID != nil})
	if err != nil {
		// A lost Redis response may follow a committed script. Do not release ownership
		// or refund quota on ambiguous transport failure; doing so would permit overbooking.
		return nil, apperror.Wrap(apperror.KindInternal, "REGISTER_QUEUE_ERROR", "failed to register queue", err)
	}
	if outcome != "OK" {
		if s.subscriptionService != nil {
			_ = s.subscriptionService.IncreaseBusinessCapacity(ctx, q.BusinessID)
		}
		if outcome == "ACTIVE_QUEUE_EXISTS" || outcome == "PHONE_ALREADY_REGISTERED" {
			return nil, apperror.New(apperror.KindConflict, outcome, "customer already has an active queue in this business")
		}
		return nil, apperror.New(apperror.KindInternal, "REGISTER_QUEUE_ERROR", "failed to register queue")
	}
	// Notification failure must not turn an accepted registration into a failed join.
	_ = s.publishQueueUpdate(ctx, q.BusinessID)
	return &RegisterCustomerQueueResult{Queue: &q, GuestID: guestID, Username: username, PhoneNumber: normalized}, nil
}

func customerQuotaError(err error) error {
	var appErr *apperror.Error
	if errors.As(err, &appErr) && (appErr.Code == "QUEUE_FULL" || appErr.Code == "BUSINESS_QUEUE_FULL") {
		return apperror.New(apperror.KindConflict, "BUSINESS_QUEUE_FULL", "the business queue is full")
	}
	return apperror.Wrap(apperror.KindInternal, "BUSINESS_QUOTA_ERROR", "failed to check business capacity", err)
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

// database layer
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
