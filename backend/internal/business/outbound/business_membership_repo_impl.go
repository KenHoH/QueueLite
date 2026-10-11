package outbound

import (
	"context"
	"errors"
	"fmt"

	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/business/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BusinessMembershipRepoImpl struct {
	db *gorm.DB
}

func NewBusinessMembershipRepo(db *gorm.DB) *BusinessRepoImpl {
	return &BusinessRepoImpl{
		db: db,
	}
}

func (r *BusinessMembershipRepoImpl) UpsertUserBusinessRelation(ctx context.Context, businessID, userID uuid.UUID, role string) error {
	relation := model.UserBusinessRelation{BusinessID: businessID, UserID: userID, Role: model.BusinessRole(role)}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "business_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "updated_at"}),
	}).Create(&relation).Error; err != nil {
		return fmt.Errorf("upsert user business relation: %w", err)
	}
	return nil
}

func (r *BusinessMembershipRepoImpl) FindMembershipUser(ctx context.Context, identifier string, byEmail bool) (*domain.MembershipUser, error) {
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
