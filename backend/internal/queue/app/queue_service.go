package app

import (
	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	"QueueLite/internal/queue/domain"
	subscriptionapp "QueueLite/internal/subscription/app"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type QueueService struct {
	repo                QueueRepo
	businessLookup      BusinessLookup
	userLookup          UserLookup
	subscriptionService *subscriptionapp.SubscriptionService
	rdt                 *redis.Client
}

type QueueSnapshotItem struct {
	QueueID   string `json:"queueId"`
	QueueName string `json:"queueName"`
}

func NewQueueService(repo QueueRepo, subscriptionService *subscriptionapp.SubscriptionService, redis *redis.Client, lookups ...any) *QueueService {
	service := &QueueService{
		repo:                repo,
		subscriptionService: subscriptionService,
		rdt:                 redis,
	}
	for _, lookup := range lookups {
		if businessLookup, ok := lookup.(BusinessLookup); ok {
			service.businessLookup = businessLookup
		}
		if userLookup, ok := lookup.(UserLookup); ok {
			service.userLookup = userLookup
		}
	}
	return service
}

type RegisterQueueByQRInput struct {
	BusinessID  uuid.UUID
	UserID      *uuid.UUID
	Username    string
	PhoneNumber string
}

type RegisterQueueByQRResult struct {
	Queue       *domain.Queue
	GuestID     *uuid.UUID
	Username    string
	PhoneNumber string
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
	return cache.RemoveWaitingQueue(ctx, s.rdt, queue.BusinessID.String(), queue.ID.String())
} // end of helper function

func (s *QueueService) ValidateBusinessExists(ctx context.Context, businessID uuid.UUID) error {
	if businessID == uuid.Nil {
		return apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
	}
	if s.businessLookup == nil {
		return nil
	}
	if _, err := s.businessLookup.GetBusiness(ctx, businessID); err != nil {
		return apperror.Wrap(apperror.KindNotFound, "BUSINESS_NOT_FOUND", "business not found", err)
	}
	return nil
}

func (s *QueueService) RegisterQueueByQR(ctx context.Context, input RegisterQueueByQRInput) (*RegisterQueueByQRResult, error) {
	if err := s.ValidateBusinessExists(ctx, input.BusinessID); err != nil {
		return nil, err
	}

	var ownerID uuid.UUID
	var guestID *uuid.UUID
	username := strings.TrimSpace(input.Username)
	phoneNumber := input.PhoneNumber

	if input.UserID != nil {
		// invalid userID that doesn't exist on database
		user, err := s.userLookup.GetUser(ctx, *input.UserID)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindUnauthorized, "USER_NOT_FOUND", "user not found", err)
		}
		ownerID = user.ID
		username = user.Username
		phoneNumber = user.PhoneNumber
	} else { // create a new temp user
		if username == "" {
			return nil, apperror.New(apperror.KindInvalid, "USERNAME_REQUIRED", "username is required")
		}
		generated := uuid.New()
		ownerID = generated
		guestID = &generated
	}

	normalizedPhone, err := NormalizeIndonesiaPhone(phoneNumber)
	if err != nil {
		return nil, err
	}

	reserved, err := cache.ReserveQueuePhone(ctx, s.rdt, input.BusinessID.String(), normalizedPhone, ownerID.String())
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "RESERVE_QUEUE_PHONE_ERROR", "failed to reserve phone number", err)
	}
	if !reserved {
		return nil, apperror.New(apperror.KindConflict, "PHONE_ALREADY_REGISTERED", "phone number already registered in this business queue")
	}

	queue, err := s.RegisterQueue(ctx, domain.Queue{
		BusinessID: input.BusinessID,
		UserID:     &ownerID,
		Priority:   false,
	})
	if err != nil {
		_ = cache.ReleaseQueuePhone(ctx, s.rdt, input.BusinessID.String(), normalizedPhone)
		return nil, err
	}

	// if user is a guest set to cache
	if guestID != nil {
		if err := cache.SetGuestQueueUser(ctx, s.rdt, cache.GuestQueueUser{
			GuestID:     guestID.String(),
			BusinessID:  input.BusinessID.String(),
			QueueID:     queue.ID.String(),
			Username:    username,
			PhoneNumber: normalizedPhone,
		}); err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "CACHE_GUEST_QUEUE_USER_ERROR", "failed to save guest queue user", err)
		}
	}

	return &RegisterQueueByQRResult{Queue: queue, GuestID: guestID, Username: username, PhoneNumber: normalizedPhone}, nil
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

	//generate the queue name
	if strings.TrimSpace(queue.Name) == "" {
		name, err := s.repo.GenerateDailyQueueName(ctx, queue.BusinessID, time.Now())
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
	queue, err := s.GetQueue(ctx, queueID)
	if err != nil {
		return err
	}
	queue.State = domain.QueueStateCancelled
	cancelledAt := time.Now()
	queue.CancelledAt = &cancelledAt
	if err := s.repo.UpdateQueue(ctx, queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "CANCEL_QUEUE_ERROR", "failed to cancel queue", err)
	}
	if err := s.syncQueueCache(ctx, *queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "CANCEL_QUEUE_CACHE_ERROR", "failed to update queue cache", err)
	}
	_ = s.publishQueueUpdate(ctx, queue.BusinessID)
	return nil
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

// func (s *QueueService) GetBusinessPublicQueueSummary(ctx context.Context, businessID uuid.UUID) (*domain.PublicQueueSummary, error) {
// 	summary, err := s.repo.GetBusinessPublicQueueSummary(ctx, businessID)
// 	if err != nil {
// 		return nil, apperror.Wrap(apperror.KindInternal, "GET_PUBLIC_QUEUE_SUMMARY_ERROR", "failed to get public queue summary", err)
// 	}
// 	return summary, nil
// }

// func (s *QueueService) GetAllQueueByBusinessFilterState(ctx context.Context, businessID uuid.UUID, state domain.QueueState) ([]domain.Queue, error) {
// 	queues, err := s.repo.GetAllQueueByBusinessFilterState(ctx, businessID, state)
// 	if err != nil {
// 		return nil, apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_QUEUES_BY_STATE_ERROR", "failed to get business queues by state", err)
// 	}
// 	return queues, nil
// }
