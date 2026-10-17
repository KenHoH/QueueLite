package app

import (
	"context"
	"errors"

	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	"QueueLite/internal/queue/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (s *QueueService) GuestOwnsQueue(ctx context.Context, queue *domain.Queue, guestID, businessID, phone string) (bool, error) {
	if queue.UserID == nil {
		return false, nil
	}
	if s.rdt == nil {
		return false, apperror.New(apperror.KindInternal, "VERIFY_QUEUE_OWNERSHIP_ERROR", "ownership cache unavailable")
	}
	customer, err := cache.GetCustomerQueue(ctx, s.rdt, queue.ID.String())
	if errors.Is(err, redis.Nil) {
		// Retain access for QR tickets created before the customer mapping existed.
		legacy, legacyErr := cache.GetGuestQueueUser(ctx, s.rdt, guestID)
		if errors.Is(legacyErr, redis.Nil) {
			return false, nil
		}
		if legacyErr != nil {
			return false, apperror.Wrap(apperror.KindInternal, "VERIFY_QUEUE_OWNERSHIP_ERROR", "failed to verify queue ownership", legacyErr)
		}
		return legacy.QueueID == queue.ID.String() && legacy.GuestID == guestID &&
			legacy.BusinessID == businessID && legacy.PhoneNumber == phone &&
			queue.UserID.String() == guestID && queue.BusinessID.String() == businessID, nil
	}
	if err != nil {
		return false, apperror.Wrap(apperror.KindInternal, "VERIFY_QUEUE_OWNERSHIP_ERROR", "failed to verify queue ownership", err)
	}
	return customer.Guest && customer.Queue.ID == queue.ID &&
		customer.Queue.UserID != nil && customer.Queue.UserID.String() == guestID &&
		queue.UserID.String() == guestID && customer.Queue.BusinessID.String() == businessID &&
		queue.BusinessID.String() == businessID && customer.PhoneNumber == phone, nil
}

func (s *QueueService) GetActiveQueueByUserAndBusiness(ctx context.Context, userID uuid.UUID, businessID uuid.UUID) (*domain.Queue, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.GetActiveQueueByUserAndBusiness(ctx, userID, businessID)
}

func (s *QueueService) GetActiveGuestQueue(ctx context.Context, guestID, businessID, phone string) (*domain.Queue, error) {
	guestUUID, err := uuid.Parse(guestID)
	if err != nil {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_GUEST_ID", "invalid guest id")
	}
	businessUUID, err := uuid.Parse(businessID)
	if err != nil {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_BUSINESS_ID", "invalid business id")
	}

	// 1. Check Redis active owner key if Redis is available
	if s.rdt != nil {
		qIDStr, err := s.rdt.Get(ctx, cache.ActiveQueueOwnerKey(businessID, guestID)).Result()
		if err == nil && qIDStr != "" {
			if qUUID, parseErr := uuid.Parse(qIDStr); parseErr == nil {
				queue, getErr := s.GetQueue(ctx, qUUID)
				if getErr == nil && queue != nil && !queue.IsTerminal() {
					owns, ownErr := s.GuestOwnsQueue(ctx, queue, guestID, businessID, phone)
					if ownErr == nil && owns {
						return queue, nil
					}
				}
			}
		}
	}

	// 2. Check Repo
	if s.repo != nil {
		queue, err := s.repo.GetActiveQueueByUserAndBusiness(ctx, guestUUID, businessUUID)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "GET_ACTIVE_QUEUE_ERROR", "failed to check active queue", err)
		}
		if queue != nil && !queue.IsTerminal() {
			owns, ownErr := s.GuestOwnsQueue(ctx, queue, guestID, businessID, phone)
			if ownErr == nil && owns {
				return queue, nil
			}
		}
	}

	return nil, nil
}

