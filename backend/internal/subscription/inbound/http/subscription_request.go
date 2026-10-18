package inbound

import (
	"errors"
	"strings"
)

type CreateSubscriptionRequest struct {
	BusinessID       string  `json:"businessId"`
	UserID           string  `json:"userId"`
	Type             string  `json:"type"`
	BusinessPlanType string  `json:"businessPlanType"`
	UserPlanType     string  `json:"userPlanType"`
	StartDate        string  `json:"startDate"`
	EndDate          *string `json:"endDate"`
	Status           string  `json:"status"`
}

type UpdateSubscriptionRequest struct {
	BusinessPlanID string `json:"businessPlanId"`
	UserPlanID     string `json:"userPlanId"`
}

type UserPlanType string

const (
	UserPlanTypeStandard UserPlanType = "standard"
	UserPlanTypePremium  UserPlanType = "premium"
)

type UpdateUserPlanRequest struct {
	UserPlanType UserPlanType `json:"user_plan_type"`
}

func (r *UpdateUserPlanRequest) Validate() error {
	trimmed := strings.TrimSpace(string(r.UserPlanType))

	if trimmed == "" {
		return errors.New("user plan type is required")
	}

	switch UserPlanType(trimmed) {
	case UserPlanTypeStandard, UserPlanTypePremium:
		return nil // Value is valid!
	default:
		return errors.New("invalid user plan type: must be 'standard' or 'premium'")
	}
}

type UpdateSubscriptionTimeRequest struct {
	StartDate string  `json:"startDate"`
	EndDate   *string `json:"endDate"`
}

type UseUserSubscriptionRequest struct {
	BusinessID string `json:"businessId"`
}

type AddUserSlotRequest struct {
	Amount int `json:"amount"`
}

type CreateSubscriptionPlanRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateSubscriptionPlanRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}
