package app

import (
	"QueueLite/internal/apperror"
	"QueueLite/internal/subscription/domain"
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
)

type exhaustedQuotaRepo struct{ SubscriptionRepo }

func (exhaustedQuotaRepo) GetBusinessSubscriptionInfo(context.Context, uuid.UUID) (*domain.BusinessSubscriptionInfo, error) {
	return &domain.BusinessSubscriptionInfo{BusinessPlan: domain.BusinessPlan{Capacity: 1}}, nil
}
func (exhaustedQuotaRepo) DecreaseBusinessCapacity(context.Context, uuid.UUID) error {
	return ErrBusinessQueueFull
}

func TestQuotaExhaustedAfterPrecheckIsConflict(t *testing.T) {
	err := NewSubscriptionService(exhaustedQuotaRepo{}).DecreaseBusinessCapacity(context.Background(), uuid.New())
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindConflict || appErr.Code != "BUSINESS_QUEUE_FULL" {
		t.Fatalf("race exhaustion: %v", err)
	}
}
