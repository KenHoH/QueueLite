package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/business/domain"
	"context"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
