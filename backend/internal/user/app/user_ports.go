package app

import (
	"context"

	"QueueLite/internal/adapter/postgres/model"
	userdomain "QueueLite/internal/user/domain"

	"github.com/google/uuid"
)

type UserRepo interface {
	CreateUser(ctx context.Context, user *userdomain.User) (*userdomain.User, error)
	UpdateUser(ctx context.Context, user *userdomain.User) error
	GetUserByName(ctx context.Context, username string) (*model.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*userdomain.User, error)
	InsertChatIDByPhoneNumber(ctx context.Context, chatID string, phoneNumber string) error
}
