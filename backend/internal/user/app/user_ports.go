package app

import (
	"QueueLite/internal/subscription/domain"
	userdomain "QueueLite/internal/user/domain"
	"context"

	"github.com/google/uuid"
)

type UserRepo interface {
	CreateUser(ctx context.Context, user *userdomain.User) (*userdomain.User, error)
	UpdateUser(ctx context.Context, user *userdomain.User) error
	GetUserByName(ctx context.Context, username string) (*userdomain.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*userdomain.User, error)
	GetUserSubscriptions(ctx context.Context, id uuid.UUID) ([]domain.Subscription, error)
}
