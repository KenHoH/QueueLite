package app

import (
	"QueueLite/internal/user/domain"
	"context"

	"github.com/google/uuid"
)

type UserRepo interface {
	CreateUser(ctx context.Context, user *domain.User) (*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) error
	GetUserByName(ctx context.Context, username string) (*domain.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error)
}
