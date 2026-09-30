package app

import (
	"QueueLite/internal/apperror"
	"QueueLite/internal/business/domain"
	"context"
	"github.com/google/uuid"
)

func (s *BusinessService) GetUserBusinesses(ctx context.Context, userID uuid.UUID) ([]domain.BusinessMembership, error) {
	items, err := s.repo.GetUserBusinesses(ctx, userID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_MY_BUSINESSES_ERROR", "failed to get your businesses", err)
	}
	return items, nil
}

func (s *BusinessService) RequireManager(ctx context.Context, userID, businessID uuid.UUID) error {
	role, err := s.repo.GetUserBusinessRole(ctx, userID, businessID)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "BUSINESS_ACCESS_ERROR", "failed to check business access", err)
	}
	if role != "owner" && role != "admin" {
		return apperror.New(apperror.KindForbidden, "BUSINESS_ACCESS_DENIED", "you cannot manage this business")
	}
	return nil
}
