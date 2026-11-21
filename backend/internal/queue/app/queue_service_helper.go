package app

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	businessdomain "QueueLite/internal/business/domain"
	counterdomain "QueueLite/internal/counter/domain"
	"QueueLite/internal/queue/domain"
	userapp "QueueLite/internal/user/app"

	"github.com/google/uuid"
)

func indexQueues(queues []domain.Queue) (map[uuid.UUID]domain.Queue, int) {
	byID := make(map[uuid.UUID]domain.Queue, len(queues))
	activeQueues := 0

	for _, queue := range queues {
		byID[queue.ID] = queue
		if isQueueBeingServed(queue.State) {
			activeQueues++
		}
	}

	return byID, activeQueues
}

func isQueueBeingServed(state domain.QueueState) bool {
	return state == domain.QueueStateCalled ||
		state == domain.QueueStateProcessing
}

func buildCounterStatus(counters []counterdomain.Counter, queuesByID map[uuid.UUID]domain.Queue) ([]CustomerCounterStatus, []CurrentlyServingQueue, int) {
	counterStatuses := make([]CustomerCounterStatus, 0, len(counters))
	serving := make([]CurrentlyServingQueue, 0, len(counters))
	activeCounters := 0

	for _, counter := range counters {
		item := CustomerCounterStatus{
			ID:             counter.ID,
			Name:           counter.Name,
			CurrentQueueID: counter.CurrentQueueID,
			State:          "idle",
		}

		if counter.CurrentQueueID != nil {
			if current, ok := queuesByID[*counter.CurrentQueueID]; ok && isQueueBeingServed(current.State) {
				name := current.Name
				item.CurrentQueueName = &name
				item.State = string(current.State)
				activeCounters++

				serving = append(serving, CurrentlyServingQueue{
					CounterID:   counter.ID,
					CounterName: counter.Name,
					QueueID:     current.ID,
					QueueName:   current.Name,
					State:       current.State,
				})
			}
		}

		counterStatuses = append(counterStatuses, item)
	}

	return counterStatuses, serving, activeCounters
}

type activeBusinessQueuesReader interface {
	GetActiveQueuesByBusiness(ctx context.Context, businessID uuid.UUID) ([]domain.Queue, error)
}

type businessCapacityReader interface {
	GetRemainingBusinessCapacity(ctx context.Context, businessID uuid.UUID) (int, error)
}

func (s *QueueService) loadCustomerQueueStatusData(ctx context.Context, queueID uuid.UUID, preloaded *domain.Queue) (*CustomerQueueStatusData, error) {
	var queue *domain.Queue
	var err error
	if preloaded != nil && preloaded.ID == queueID {
		copyQueue := *preloaded
		queue = &copyQueue
	} else {
		queue, err = s.GetQueue(ctx, queueID)
		if err != nil {
			return nil, err
		}
	}

	business, err := s.queueBusinessRepo.GetBusiness(ctx, queue.BusinessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_ERROR", "failed to get business", err)
	}

	counters, err := s.queueCounterRepo.ListBusinessCounters(ctx, queue.BusinessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_COUNTERS_ERROR", "failed to get counters", err)
	}

	var queues []domain.Queue
	if activeReader, ok := s.repo.(activeBusinessQueuesReader); ok {
		queues, err = activeReader.GetActiveQueuesByBusiness(ctx, queue.BusinessID)
	} else {
		queues, err = s.repo.GetAllQueueByBusiness(ctx, queue.BusinessID)
	}
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_QUEUES_ERROR", "failed to get business queues", err)
	}

	waiting, err := s.GetWaitingQueueSnapshot(ctx, queue.BusinessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_WAITING_QUEUE_ERROR", "failed to get waiting queues", err)
	}

	return &CustomerQueueStatusData{
		queue:    queue,
		business: business,
		counters: counters,
		queues:   queues,
		waiting:  waiting,
	}, nil
}

func (s *QueueService) validateCustomerQueueBusiness(ctx context.Context, businessID uuid.UUID) (*businessdomain.Business, error) {
	if businessID == uuid.Nil {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
	}
	if s.queueBusinessRepo == nil {
		return nil, apperror.New(apperror.KindInternal, "BUSINESS_LOOKUP_UNAVAILABLE", "business lookup unavailable")
	}

	business, err := s.queueBusinessRepo.GetBusiness(ctx, businessID)
	if err != nil {
		if errors.Is(err, businessdomain.ErrBusinessNotFound) {
			return nil, apperror.Wrap(apperror.KindNotFound, "BUSINESS_NOT_FOUND", "business not found", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_ERROR", "failed to get business", err)
	}

	if !business.Operational {
		return nil, apperror.New(apperror.KindConflict, "BUSINESS_UNAVAILABLE", "business is unavailable")
	}
	return business, nil
}

func (s *QueueService) loadRegisteredCustomer(ctx context.Context, input RegisterCustomerQueueInput) (QueueCustomer, error) {
	if input.AuthenticatedUser != nil && input.AuthenticatedUser.ID == *input.UserID {
		return QueueCustomer{
			OwnerID:     input.AuthenticatedUser.ID,
			Username:    input.AuthenticatedUser.Username,
			PhoneNumber: input.AuthenticatedUser.PhoneNumber,
		}, nil
	}
	if s.queueUserRepo == nil {
		return QueueCustomer{}, apperror.New(apperror.KindInternal, "USER_LOOKUP_UNAVAILABLE", "user lookup unavailable")
	}

	user, err := s.queueUserRepo.GetUser(ctx, *input.UserID)
	if err != nil {
		if errors.Is(err, userapp.ErrUserNotFound) {
			return QueueCustomer{}, apperror.Wrap(apperror.KindUnauthorized, "USER_NOT_FOUND", "user not found", err)
		}
		return QueueCustomer{}, apperror.Wrap(apperror.KindInternal, "GET_USER_ERROR", "failed to get user", err)
	}
	return QueueCustomer{
		OwnerID:     user.ID,
		Username:    user.Username,
		PhoneNumber: user.PhoneNumber,
	}, nil
}

func (s *QueueService) resolveQueueCustomer(ctx context.Context, input RegisterCustomerQueueInput) (QueueCustomer, error) {
	var customer QueueCustomer
	var err error

	if input.UserID != nil {
		customer, err = s.loadRegisteredCustomer(ctx, input)
		if err != nil {
			return QueueCustomer{}, err
		}
	} else {
		ownerID := uuid.New()
		customer = QueueCustomer{
			OwnerID:     ownerID,
			GuestID:     &ownerID,
			Username:    input.Username,
			PhoneNumber: input.PhoneNumber,
		}
	}

	customer.Username = strings.TrimSpace(customer.Username)
	if customer.Username == "" {
		return QueueCustomer{}, apperror.New(apperror.KindInvalid, "USERNAME_REQUIRED", "username is required")
	}
	if strings.TrimSpace(customer.PhoneNumber) == "" {
		return QueueCustomer{}, apperror.New(apperror.KindInvalid, "PHONE_NUMBER_REQUIRED", "phone number is required")
	}

	normalized, err := NormalizeIndonesiaPhone(customer.PhoneNumber)
	if err != nil {
		return QueueCustomer{}, err
	}
	customer.PhoneNumber = normalized
	return customer, nil
}

func (s *QueueService) allocateDailyQueueName(ctx context.Context, businessID uuid.UUID, now time.Time) (string, error) {
	var floor int64
	hasCounter, counterErr := cache.HasDailyQueueCounter(ctx, s.rdt, businessID.String(), now)
	if counterErr != nil || !hasCounter {
		var err error
		floor, err = s.repo.GetDailyQueueNumberFloor(ctx, businessID, now)
		if err != nil {
			return "", apperror.Wrap(apperror.KindInternal, "GENERATE_QUEUE_NAME_ERROR", "failed to allocate queue number", err)
		}
	}

	name, err := cache.GenerateDailyQueueName(ctx, s.rdt, businessID.String(), now, floor)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "GENERATE_QUEUE_NAME_ERROR", "failed to allocate queue number", err)
	}
	return name, nil
}

func (s *QueueService) prepareCustomerQueue(ctx context.Context, businessID uuid.UUID, customer QueueCustomer) (domain.Queue, error) {
	active, err := s.repo.GetActiveQueueByUserAndBusiness(ctx, customer.OwnerID, businessID)
	if err != nil {
		return domain.Queue{}, apperror.Wrap(apperror.KindInternal, "GET_ACTIVE_QUEUE_ERROR", "failed to check active queue", err)
	}
	if active != nil {
		return domain.Queue{}, apperror.New(apperror.KindConflict, "ACTIVE_QUEUE_EXISTS", "user already has active queue in this business")
	}

	now := time.Now().UTC()
	name, err := s.allocateDailyQueueName(ctx, businessID, now)
	if err != nil {
		return domain.Queue{}, err
	}

	return domain.Queue{
		ID:         uuid.New(),
		BusinessID: businessID,
		UserID:     &customer.OwnerID,
		Name:       name,
		State:      domain.QueueStateWaiting,
		Priority:   false,
		CreatedAt:  now,
	}, nil
}

func (s *QueueService) registerCustomerQueueAtomically(ctx context.Context, queue domain.Queue, customer QueueCustomer) error {
	payload := cache.CustomerQueue{
		Queue:       queue,
		Username:    customer.Username,
		PhoneNumber: customer.PhoneNumber,
		Guest:       customer.GuestID != nil,
	}

	// Fast path: enforce daily business capacity atomically inside Redis Lua, seeding from
	// Postgres once per business/UTC day on cache miss and syncing decrements asynchronously.
	if s.rdt != nil {
		if capReader, ok := s.subscriptionService.(businessCapacityReader); ok {
			return s.registerCustomerQueueWithRedisCapacity(ctx, queue, payload, capReader)
		}
	}

	return s.registerCustomerQueueWithDBCapacity(ctx, queue, payload)
}

func (s *QueueService) registerCustomerQueueWithDBCapacity(ctx context.Context, queue domain.Queue, payload cache.CustomerQueue) error {
	quotaDecreased := false
	if s.subscriptionService != nil {
		if err := s.subscriptionService.DecreaseBusinessCapacity(ctx, queue.BusinessID); err != nil {
			return CustomerQuotaError(err)
		}
		quotaDecreased = true
	}

	outcome, err := cache.EnqueueCustomerQueue(ctx, s.rdt, payload)
	if err != nil {
		if quotaDecreased {
			s.refundBusinessCapacity(ctx, queue.BusinessID, queue.ID)
		}
		return apperror.Wrap(apperror.KindInternal, "REGISTER_QUEUE_ERROR", "failed to register queue", err)
	}
	if outcome == "OK" {
		return nil
	}

	if quotaDecreased {
		s.refundBusinessCapacity(ctx, queue.BusinessID, queue.ID)
	}
	return mapEnqueueOutcomeError(outcome)
}

func (s *QueueService) refundBusinessCapacity(ctx context.Context, businessID uuid.UUID, queueID uuid.UUID) {
	if s.subscriptionService == nil {
		return
	}
	refundCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if refundErr := s.subscriptionService.IncreaseBusinessCapacity(refundCtx, businessID); refundErr != nil {
		log.Printf("[QUEUE QUOTA ERROR] failed to refund business capacity for business=%s queue=%s: %v", businessID, queueID, refundErr)
	}
}

func (s *QueueService) registerCustomerQueueWithRedisCapacity(
	ctx context.Context,
	queue domain.Queue,
	payload cache.CustomerQueue,
	capReader businessCapacityReader,
) error {
	for attempt := 0; attempt < 2; attempt++ {
		outcome, err := cache.EnqueueCustomerQueueWithCapacity(ctx, s.rdt, payload, queue.CreatedAt)
		if err != nil {
			return apperror.Wrap(apperror.KindInternal, "REGISTER_QUEUE_ERROR", "failed to register queue", err)
		}
		if outcome == "OK" {
			return nil
		}
		if outcome == "CAPACITY_NOT_SEEDED" && attempt == 0 {
			remaining, capErr := capReader.GetRemainingBusinessCapacity(ctx, queue.BusinessID)
			if capErr != nil {
				return CustomerQuotaError(capErr)
			}
			if seedErr := cache.SeedBusinessCapacity(ctx, s.rdt, queue.BusinessID.String(), queue.CreatedAt, remaining); seedErr != nil {
				return apperror.Wrap(apperror.KindInternal, "REGISTER_QUEUE_ERROR", "failed to seed business capacity", seedErr)
			}
			continue
		}
		return mapEnqueueOutcomeError(outcome)
	}
	return apperror.New(apperror.KindInternal, "REGISTER_QUEUE_ERROR", "failed to register queue")
}

func mapEnqueueOutcomeError(outcome string) error {
	switch outcome {
	case "ACTIVE_QUEUE_EXISTS", "PHONE_ALREADY_REGISTERED":
		return apperror.New(apperror.KindConflict, outcome, "customer already has an active queue in this business")
	case "BUSINESS_QUEUE_FULL":
		return apperror.New(apperror.KindConflict, "BUSINESS_QUEUE_FULL", "the business queue is full")
	default:
		return apperror.New(apperror.KindInternal, "REGISTER_QUEUE_ERROR", "failed to register queue")
	}
}
