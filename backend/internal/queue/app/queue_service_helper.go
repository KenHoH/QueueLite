package app

import (
	"context"
	"errors"
	"strings"
	"time"

	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	businessdomain "QueueLite/internal/business/domain"
	counterdomain "QueueLite/internal/counter/domain"
	userapp "QueueLite/internal/user/app"

	"QueueLite/internal/queue/domain"

	"github.com/google/uuid"
)

func indexQueues(queues []domain.Queue) (map[uuid.UUID]domain.Queue, int) {
	byID := make(map[uuid.UUID]domain.Queue, len(queues))
	activeQueues := 0

	for _, queue := range queues {
		byID[queue.ID] = queue

		if queue.State == domain.QueueStateCalled || queue.State == domain.QueueStateProcessing {
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
			current, ok := queuesByID[*counter.CurrentQueueID]

			if ok && isQueueBeingServed(current.State) {
				name := current.Name

				item.CurrentQueueName = &name
				item.State = string(current.State)

				activeCounters++

				serving = append(
					serving,
					CurrentlyServingQueue{
						CounterID:   counter.ID,
						CounterName: counter.Name,
						QueueID:     current.ID,
						QueueName:   current.Name,
						State:       current.State,
					},
				)
			}
		}

		counterStatuses = append(counterStatuses, item)
	}

	return counterStatuses, serving, activeCounters
}

func (s *QueueService) loadCustomerQueueStatusData(ctx context.Context, queueID uuid.UUID) (*CustomerQueueStatusData, error) {
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

func (s *QueueService) resolveQueueCustomer(ctx context.Context, input RegisterCustomerQueueInput) (QueueCustomer, error) {
	customer := QueueCustomer{
		OwnerID:     uuid.New(),
		Username:    strings.TrimSpace(input.Username),
		PhoneNumber: input.PhoneNumber,
	}

	if input.UserID != nil {
		if s.queueUserRepo == nil {
			return QueueCustomer{}, apperror.New(
				apperror.KindInternal,
				"USER_LOOKUP_UNAVAILABLE",
				"user lookup unavailable",
			)
		}

		user, err := s.queueUserRepo.GetUser(ctx, *input.UserID)
		if err != nil {
			if errors.Is(err, userapp.ErrUserNotFound) {
				return QueueCustomer{}, apperror.Wrap(
					apperror.KindUnauthorized,
					"USER_NOT_FOUND",
					"user not found",
					err,
				)
			}

			return QueueCustomer{}, apperror.Wrap(
				apperror.KindInternal,
				"GET_USER_ERROR",
				"failed to get user",
				err,
			)
		}

		customer.OwnerID = user.ID
		customer.Username = user.Username
		customer.PhoneNumber = user.PhoneNumber
	} else {
		customer.GuestID = &customer.OwnerID
	}

	customer.Username = strings.TrimSpace(customer.Username)

	if customer.Username == "" {
		return QueueCustomer{}, apperror.New(
			apperror.KindInvalid,
			"USERNAME_REQUIRED",
			"username is required",
		)
	}

	if strings.TrimSpace(customer.PhoneNumber) == "" {
		return QueueCustomer{}, apperror.New(
			apperror.KindInvalid,
			"PHONE_NUMBER_REQUIRED",
			"phone number is required",
		)
	}

	normalized, err := NormalizeIndonesiaPhone(customer.PhoneNumber)
	if err != nil {
		return QueueCustomer{}, err
	}

	customer.PhoneNumber = normalized

	return customer, nil
}

func (s *QueueService) prepareCustomerQueue(ctx context.Context, businessID uuid.UUID, customer QueueCustomer) (domain.Queue, error) {
	active, err := s.repo.GetActiveQueueByUserAndBusiness(
		ctx,
		customer.OwnerID,
		businessID,
	)
	if err != nil {
		return domain.Queue{}, apperror.Wrap(
			apperror.KindInternal,
			"GET_ACTIVE_QUEUE_ERROR",
			"failed to check active queue",
			err,
		)
	}

	if active != nil {
		return domain.Queue{}, apperror.New(
			apperror.KindConflict,
			"ACTIVE_QUEUE_EXISTS",
			"user already has active queue in this business",
		)
	}

	now := time.Now().UTC()

	queue := domain.Queue{
		ID:         uuid.New(),
		BusinessID: businessID,
		UserID:     &customer.OwnerID,
		State:      domain.QueueStateWaiting,
		Priority:   false,
		CreatedAt:  now,
	}

	floor, err := s.repo.GetDailyQueueNumberFloor(
		ctx,
		businessID,
		now,
	)
	if err != nil {
		return domain.Queue{}, apperror.Wrap(
			apperror.KindInternal,
			"GENERATE_QUEUE_NAME_ERROR",
			"failed to allocate queue number",
			err,
		)
	}

	queue.Name, err = cache.GenerateDailyQueueName(
		ctx,
		s.rdt,
		businessID.String(),
		now,
		floor,
	)
	if err != nil {
		return domain.Queue{}, apperror.Wrap(
			apperror.KindInternal,
			"GENERATE_QUEUE_NAME_ERROR",
			"failed to allocate queue number",
			err,
		)
	}

	return queue, nil
}

func (s *QueueService) registerCustomerQueueAtomically(ctx context.Context, queue domain.Queue, customer QueueCustomer) error {
	if s.subscriptionService != nil {
		if err := s.subscriptionService.DecreaseBusinessCapacity(
			ctx,
			queue.BusinessID,
		); err != nil {
			return CustomerQuotaError(err)
		}
	}

	outcome, err := cache.EnqueueCustomerQueue(
		ctx,
		s.rdt,
		cache.CustomerQueue{
			Queue:       queue,
			Username:    customer.Username,
			PhoneNumber: customer.PhoneNumber,
			Guest:       customer.GuestID != nil,
		},
	)
	if err != nil {
		return apperror.Wrap(
			apperror.KindInternal,
			"REGISTER_QUEUE_ERROR",
			"failed to register queue",
			err,
		)
	}

	if outcome == "OK" {
		return nil
	}

	if s.subscriptionService != nil {
		if refundErr := s.subscriptionService.IncreaseBusinessCapacity(ctx, queue.BusinessID); refundErr != nil {
			// Log this failure and reconcile the quota.
			// Do not silently lose track of the inconsistency.
		}
	}

	switch outcome {
	case "ACTIVE_QUEUE_EXISTS", "PHONE_ALREADY_REGISTERED":
		return apperror.New(
			apperror.KindConflict,
			outcome,
			"customer already has an active queue in this business",
		)
	default:
		return apperror.New(
			apperror.KindInternal,
			"REGISTER_QUEUE_ERROR",
			"failed to register queue",
		)
	}
}
