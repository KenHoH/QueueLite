package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/user/app"
	"QueueLite/internal/user/domain"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepoImpl struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepoImpl {
	return &UserRepoImpl{
		db: db,
	}
}

func (u *UserRepoImpl) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	record := toUserRecord(user)
	if err := u.db.
		WithContext(ctx).
		Create(&record).
		Error; err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	user.ID = record.ID
	return user, nil
}

func (u *UserRepoImpl) UpdateUser(ctx context.Context, user *domain.User) error {
	result := u.db.
		WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]any{
			"username":     user.Username,
			"phone_number": user.PhoneNumber,
			"email":        user.Email,
		})

	if result.Error != nil {
		return fmt.Errorf("update user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrUserNotFound, user.ID)
	}

	return nil
}

func (u *UserRepoImpl) GetUserByName(ctx context.Context, username string) (*domain.User, error) {
	var record model.User
	err := u.db.
		WithContext(ctx).
		Where("username = ?", username).
		Take(&record).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %s", app.ErrUserNotFound, username)
	}

	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return toDomainUser(&record), nil
}
func (u *UserRepoImpl) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var record model.User
	err := u.db.
		WithContext(ctx).
		Where("id = ?", id).
		Take(&record).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %s", app.ErrUserNotFound, id)
	}

	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return toDomainUser(&record), nil
}
