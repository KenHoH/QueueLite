package inbound

import (
	"QueueLite/internal/subscription/domain"
	"time"

	"github.com/google/uuid"
)

type SubscriptionResponse struct {
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

type BusinessPlanResponse struct {
	ID                  string `json:"id"`
	BusinessPlanType    string `json:"businessPlanType"`
	Description         string `json:"description"`
	Price               int64  `json:"price"`
	Capacity            int    `json:"capacity"`
	Analysis            bool   `json:"analysis"`
	Insight             bool   `json:"insight"`
	PrioritySupport     bool   `json:"prioritySupport"`
	LastCapacityResetAt string `json:"lastCapacityResetAt"`
}

type UserPlanResponse struct {
	ID               string `json:"id"`
	UserPlanType     string `json:"userPlanType"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Price            int64  `json:"price"`
	Slots            int    `json:"slots"`
	LastSlotsResetAt string `json:"lastSlotsResetAt"`
}

type BusinessSubscriptionInfoResponse struct {
	Subscription SubscriptionResponse `json:"subscription"`
	BusinessPlan BusinessPlanResponse `json:"businessPlan"`
}

type UserSubscriptionInfoResponse struct {
	Subscription SubscriptionResponse `json:"subscription"`
	UserPlan     UserPlanResponse     `json:"userPlan"`
}

type UseUserSubscriptionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type SubscriptionPlanResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SubscriptionCursorResponse struct {
	CreatedAt string `json:"createdAt"`
	ID        string `json:"id"`
}

func NewSubscriptionResponse(subscription *domain.Subscription) SubscriptionResponse {
	var endDate *string
	if subscription.EndDate != nil {
		value := subscription.EndDate.Format(time.RFC3339)
		endDate = &value
	}

	return SubscriptionResponse{
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

func NewSubscriptionResponses(subscriptions []domain.Subscription) []SubscriptionResponse {
	responses := make([]SubscriptionResponse, 0, len(subscriptions))
	for i := range subscriptions {
		responses = append(responses, NewSubscriptionResponse(&subscriptions[i]))
	}
	return responses
}

func NewBusinessPlanResponse(plan *domain.BusinessPlan) BusinessPlanResponse {
	return BusinessPlanResponse{
		ID:                  plan.ID.String(),
		BusinessPlanType:    string(plan.BusinessPlanType),
		Description:         plan.Description,
		Price:               plan.Price,
		Capacity:            plan.Capacity,
		Analysis:            plan.Analysis,
		Insight:             plan.Insight,
		PrioritySupport:     plan.PrioritySupport,
		LastCapacityResetAt: plan.LastCapacityResetAt.Format(time.RFC3339),
	}
}

func NewUserPlanResponse(plan *domain.UserPlan) UserPlanResponse {
	return UserPlanResponse{
		ID:               plan.ID.String(),
		UserPlanType:     string(plan.UserPlanType),
		Name:             plan.Name,
		Description:      plan.Description,
		Price:            plan.Price,
		Slots:            plan.Slots,
		LastSlotsResetAt: plan.LastSlotsResetAt.Format(time.RFC3339),
	}
}

func NewBusinessSubscriptionInfoResponse(info *domain.BusinessSubscriptionInfo) BusinessSubscriptionInfoResponse {
	return BusinessSubscriptionInfoResponse{
		Subscription: NewSubscriptionResponse(&info.Subscription),
		BusinessPlan: NewBusinessPlanResponse(&info.BusinessPlan),
	}
}

func NewUserSubscriptionInfoResponse(info *domain.UserSubscriptionInfo) UserSubscriptionInfoResponse {
	return UserSubscriptionInfoResponse{
		Subscription: NewSubscriptionResponse(&info.Subscription),
		UserPlan:     NewUserPlanResponse(&info.UserPlan),
	}
}

func NewUseUserSubscriptionResponse(result *domain.UseUserSubscriptionResult) UseUserSubscriptionResponse {
	return UseUserSubscriptionResponse{Success: result.Success, Message: result.Message}
}

func NewSubscriptionPlanResponse(plan *domain.SubscriptionPlan) SubscriptionPlanResponse {
	return SubscriptionPlanResponse{
		ID:          plan.ID.String(),
		Name:        plan.Name,
		Description: plan.Description,
	}
}

func NewSubscriptionCursorResponse(cursor *domain.SubscriptionCursor) *SubscriptionCursorResponse {
	if cursor == nil {
		return nil
	}
	return &SubscriptionCursorResponse{
		CreatedAt: cursor.CreatedAt.Format(time.RFC3339),
		ID:        cursor.ID.String(),
	}
}

func uuidString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	value := id.String()
	return &value
}
