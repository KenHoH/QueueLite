package app

import "errors"

var (
	ErrBusinessQueueFull         = errors.New("business queue full")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionPlanNotFound  = errors.New("subscription plan not found")
	ErrBusinessPlanNotFound      = errors.New("business plan not found")
	ErrUserPlanNotFound          = errors.New("user plan not found")
)
