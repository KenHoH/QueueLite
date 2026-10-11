package outbound

import (
	"context"
	"errors"
	"fmt"

	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/business/app"
	"QueueLite/internal/business/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BusinessRepoImpl struct {
	db *gorm.DB
}

func NewBusinessRepo(db *gorm.DB) *BusinessRepoImpl {
	return &BusinessRepoImpl{
		db: db,
	}
}

func (r *BusinessRepoImpl) CreateBusiness(ctx context.Context, business *domain.Business) (*domain.Business, error) {
	record := toBusinessRecord(business)
	if err := r.db.
		WithContext(ctx).
		Create(&record).
		Error; err != nil {
		return nil, fmt.Errorf("create business: %w", err)
	}

	business.ID = record.ID
	return business, nil
}

func (r *BusinessRepoImpl) CreateUserBusinessRelation(ctx context.Context, businessID uuid.UUID, userID uuid.UUID, role string) error {
	record := model.UserBusinessRelation{BusinessID: businessID, UserID: userID, Role: model.BusinessRole(role)}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("create user business relation: %w", err)
	}
	return nil
}

func (r *BusinessRepoImpl) UpdateBusiness(ctx context.Context, businessId uuid.UUID, input *domain.UpdateBusiness) error {
	updates := map[string]any{}

	if input.Name != nil {
		updates["business_name"] = *input.Name
	}

	if input.Description != nil {
		updates["business_description"] = *input.Description
	}

	if input.Location != nil {
		updates["business_location"] = *input.Location
	}

	if input.Operational != nil {
		updates["business_operational"] = *input.Operational
	}

	if input.OpenTime != nil {
		updates["business_open_time"] = *input.OpenTime
	}

	if input.CloseTime != nil {
		updates["business_close_time"] = *input.CloseTime
	}

	if input.Email != nil {
		updates["business_email"] = *input.Email
	}

	if input.PhoneNumber != nil {
		updates["business_phone_number"] = *input.PhoneNumber
	}

	result := r.db.
		WithContext(ctx).
		Model(&model.Business{}).
		Where("id = ?", businessId).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf(
			"update business: %w",
			result.Error,
		)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf(
			"%w: %s",
			app.ErrBusinessNotFound,
			businessId,
		)
	}

	return nil
}

func (r *BusinessRepoImpl) GetBusiness(ctx context.Context, id uuid.UUID) (*domain.Business, error) {
	var record model.Business
	err := r.db.
		WithContext(ctx).
		Where("id = ?", id).
		Take(&record).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %s", app.ErrBusinessNotFound, id)
	}

	if err != nil {
		return nil, fmt.Errorf("get business: %w", err)
	}

	return toDomainBusiness(&record), nil
}

func (r *BusinessRepoImpl) GetAllBusiness(
	ctx context.Context,
	cursor *domain.BusinessCursor,
	limit int,
) ([]domain.Business, *domain.BusinessCursor, error) {
	var businesses []model.Business

	query := r.db.
		WithContext(ctx).
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit)

	if cursor != nil {
		query = query.Where(
			`
            created_at < ?
            OR
            (
                created_at = ?
                AND id < ?
            )
        `,
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}

	if err := query.Find(&businesses).Error; err != nil {
		return nil, nil, err
	}

	var nextCursor *domain.BusinessCursor

	if len(businesses) > 0 {

		last := businesses[len(businesses)-1]

		nextCursor = &domain.BusinessCursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		}
	}

	return toDomainBusinesses(businesses), nextCursor, nil
}

func (r *BusinessRepoImpl) SearchBusiness(
	ctx context.Context,
	searchQuery string,
) ([]domain.Business, error) {
	var businesses []model.Business

	err := r.db.
		WithContext(ctx).
		Where(
			"business_name ILIKE ?",
			"%"+searchQuery+"%",
		).
		Find(&businesses).
		Error
	if err != nil {
		return nil, err
	}

	return toDomainBusinesses(businesses), nil
}

func (r *BusinessRepoImpl) DeleteBusiness(ctx context.Context, id uuid.UUID) error {
	result := r.db.
		WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.Business{})

	if result.Error != nil {
		return fmt.Errorf("delete business: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrBusinessNotFound, id)
	}

	return nil
}

func (r *BusinessRepoImpl) ListBusinessMembers(ctx context.Context, businessID uuid.UUID) ([]domain.BusinessMember, error) {
	var result []domain.BusinessMember
	err := r.db.WithContext(ctx).Table("user_business_relations AS relations").
		Select("relations.user_id, users.username, relations.role").
		Joins("JOIN users ON users.id = relations.user_id").
		Where("relations.business_id = ?", businessID).Order("users.username ASC, relations.user_id ASC").Scan(&result).Error
	return result, err
}

func (r *BusinessRepoImpl) GetUserBusinesses(ctx context.Context, userID uuid.UUID) ([]domain.BusinessMembership, error) {
	var relations []model.UserBusinessRelation
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at ASC, business_id ASC").Preload("Business").Find(&relations).Error; err != nil {
		return nil, err
	}
	items := make([]domain.BusinessMembership, 0, len(relations))
	for _, relation := range relations {
		items = append(items, domain.BusinessMembership{Business: *toDomainBusiness(&relation.Business), Role: string(relation.Role)})
	}
	return items, nil
}

func (r *BusinessRepoImpl) FindMembershipUser(ctx context.Context, identifier string, byEmail bool) (*domain.MembershipUser, error) {
	var user domain.MembershipUser
	query := r.db.WithContext(ctx).Model(&model.User{}).Select("id, username")
	if byEmail {
		query = query.Where("LOWER(email) = LOWER(?)", identifier)
	} else {
		query = query.Where("username = ?", identifier)
	}
	if err := query.Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %s", domain.ErrMembershipUserNotFound, identifier)
		}
		return nil, fmt.Errorf("find membership user: %w", err)
	}
	return &user, nil
}

func (r *BusinessRepoImpl) GetUserBusinessRole(ctx context.Context, userID, businessID uuid.UUID) (string, error) {
	var relation model.UserBusinessRelation
	err := r.db.WithContext(ctx).Where("user_id = ? AND business_id = ?", userID, businessID).Take(&relation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(relation.Role), nil
}
