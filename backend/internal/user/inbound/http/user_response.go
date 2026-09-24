package inbound

import (
	subscriptiondomain "QueueLite/internal/subscription/domain"
	"QueueLite/internal/user/domain"
	"time"

	"github.com/google/uuid"
)

type UserResponse struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	PhoneNumber string  `json:"phonenumber"`
	Email       *string `json:"email,omitempty"`
}

type UserSubscriptionResponse struct {
	ID             string  `json:"id"`
	BusinessID     *string `json:"businessId,omitempty"`
	UserID         *string `json:"userId,omitempty"`
	BusinessPlanID *string `json:"businessPlanId,omitempty"`
	UserPlanID     *string `json:"userPlanId,omitempty"`
	Type           string  `json:"type"`
	StartDate      string  `json:"startDate"`
	EndDate        *string `json:"endDate,omitempty"`
	Status         string  `json:"status"`
}

func NewUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:          user.ID.String(),
		Username:    user.Username,
		PhoneNumber: user.PhoneNumber,
		Email:       user.Email,
	}
}

func NewUserSubscriptionResponse(subscription *subscriptiondomain.Subscription) UserSubscriptionResponse {
	var endDate *string
	if subscription.EndDate != nil {
		value := subscription.EndDate.Format(time.RFC3339)
		endDate = &value
	}
	return UserSubscriptionResponse{
		ID:             subscription.ID.String(),
		BusinessID:     uuidString(subscription.BusinessID),
		UserID:         uuidString(subscription.UserID),
		BusinessPlanID: uuidString(subscription.BusinessPlanID),
		UserPlanID:     uuidString(subscription.UserPlanID),
		Type:           string(subscription.Type),
		StartDate:      subscription.StartDate.Format(time.RFC3339),
		EndDate:        endDate,
		Status:         string(subscription.Status),
	}
}

func NewUserSubscriptionResponses(subscriptions []subscriptiondomain.Subscription) []UserSubscriptionResponse {
	responses := make([]UserSubscriptionResponse, 0, len(subscriptions))
	for i := range subscriptions {
		responses = append(responses, NewUserSubscriptionResponse(&subscriptions[i]))
	}
	return responses
}

func uuidString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	value := id.String()
	return &value
}
