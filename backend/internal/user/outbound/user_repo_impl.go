package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	subscriptiondomain "QueueLite/internal/subscription/domain"
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
	updates := map[string]any{
		"username":     user.Username,
		"phone_number": user.PhoneNumber,
		"email":        user.Email,
	}
	if user.Password != "" {
		updates["password"] = user.Password
	}

	result := u.db.
		WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", user.ID).
		Updates(updates)

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
func (u *UserRepoImpl) GetUserSubscriptions(ctx context.Context, id uuid.UUID) ([]subscriptiondomain.Subscription, error) {
	var records []model.Subscription
	if err := u.db.
		WithContext(ctx).
		Joins("JOIN user_business_relations ON user_business_relations.business_id = subscriptions.business_id").
		Where("user_business_relations.user_id = ?", id).
		Order("subscriptions.created_at DESC").
		Find(&records).
		Error; err != nil {
		return nil, fmt.Errorf("get user subscriptions: %w", err)
	}

	subscriptions := make([]subscriptiondomain.Subscription, 0, len(records))
	for i := range records {
		subscriptions = append(subscriptions, subscriptiondomain.Subscription{
			ID:                 records[i].ID,
			BusinessID:         records[i].BusinessID,
			SubscriptionPlanID: records[i].SubscriptionPlanID,
			Type:               subscriptiondomain.SubscriptionType(records[i].Type),
			StartDate:          records[i].StartDate,
			EndDate:            records[i].EndDate,
			Status:             subscriptiondomain.SubscriptionStatus(records[i].Status),
			CreatedAt:          records[i].CreatedAt,
		})
	}
	return subscriptions, nil
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
