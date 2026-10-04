package app

import (
	"context"
	"errors"

	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	"QueueLite/internal/queue/domain"

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
