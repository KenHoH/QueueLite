package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/user/domain"
)

func toUserRecord(user *domain.User) model.User {
	var password *string
	if user.Password != "" {
		password = &user.Password
	}

	return model.User{
		ID:          user.ID,
		Username:    user.Username,
		PhoneNumber: user.PhoneNumber,
		Password:    password,
		Email:       user.Email,
	}
}

func toDomainUser(record *model.User) *domain.User {
	var password string
	if record.Password != nil {
		password = *record.Password
	}

	return &domain.User{
		ID:          record.ID,
		Username:    record.Username,
		PhoneNumber: record.PhoneNumber,
		Password:    password,
		Email:       record.Email,
	}
}
