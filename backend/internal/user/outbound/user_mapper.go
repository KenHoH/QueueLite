package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/user/domain"
)

func toUserRecord(user *domain.User) model.User {
	return model.User{
		ID:          user.ID,
		Username:    user.Username,
		PhoneNumber: user.PhoneNumber,
		Email:       user.Email,
	}
}

func toDomainUser(record *model.User) *domain.User {
	return &domain.User{
		ID:          record.ID,
		Username:    record.Username,
		PhoneNumber: record.PhoneNumber,
		Email:       record.Email,
	}
}
