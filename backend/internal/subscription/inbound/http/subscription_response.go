package inbound

import (
	"QueueLite/internal/subscription/domain"
	"time"
)

type SubscriptionResponse struct {
	ID                 string  `json:"id"`
	BusinessID         string  `json:"businessId"`
	SubscriptionPlanID string  `json:"subscriptionPlanId"`
	Type               string  `json:"type"`
	StartDate          string  `json:"startDate"`
	EndDate            *string `json:"endDate,omitempty"`
	Status             string  `json:"status"`
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
		ID:                 subscription.ID.String(),
		BusinessID:         subscription.BusinessID.String(),
		SubscriptionPlanID: subscription.SubscriptionPlanID.String(),
		Type:               string(subscription.Type),
		StartDate:          subscription.StartDate.Format(time.RFC3339),
		EndDate:            endDate,
		Status:             string(subscription.Status),
	}
}

func NewSubscriptionResponses(subscriptions []domain.Subscription) []SubscriptionResponse {
	responses := make([]SubscriptionResponse, 0, len(subscriptions))
	for i := range subscriptions {
		responses = append(responses, NewSubscriptionResponse(&subscriptions[i]))
	}
	return responses
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
