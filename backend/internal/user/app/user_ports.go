package app

import (
	"QueueLite/internal/adapter/postgres/model"
	userdomain "QueueLite/internal/user/domain"
	"context"

	"github.com/google/uuid"
)

type UserRepo interface {
	CreateUser(ctx context.Context, user *userdomain.User) (*userdomain.User, error)
	UpdateUser(ctx context.Context, user *userdomain.User) error
	GetUserByName(ctx context.Context, username string) (*model.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*userdomain.User, error)
}
