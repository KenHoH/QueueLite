package app

import (
	"QueueLite/internal/business/domain"
	"context"

	"github.com/google/uuid"
)

type BusinessRepo interface {
	CreateBusiness(ctx context.Context, business *domain.Business) (*domain.Business, error)
	CreateUserBusinessRelation(ctx context.Context, businessID uuid.UUID, userID uuid.UUID, role string) error
	UpdateBusiness(ctx context.Context, businessID uuid.UUID, input *domain.UpdateBusiness) error
	GetBusiness(ctx context.Context, id uuid.UUID) (*domain.Business, error)
	GetAllBusiness(ctx context.Context, cursor *domain.BusinessCursor, limit int) ([]domain.Business, *domain.BusinessCursor, error)
	SearchBusiness(ctx context.Context, searchQuery string) ([]domain.Business, error)
	DeleteBusiness(ctx context.Context, id uuid.UUID) error
}
