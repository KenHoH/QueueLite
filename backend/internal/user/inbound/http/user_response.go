package inbound

import "QueueLite/internal/user/domain"

type UserResponse struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	PhoneNumber string  `json:"phonenumber"`
	Email       *string `json:"email,omitempty"`
}

func NewUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:          user.ID.String(),
		Username:    user.Username,
		PhoneNumber: user.PhoneNumber,
		Email:       user.Email,
	}
}
