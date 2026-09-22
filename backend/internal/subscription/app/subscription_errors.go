package app

import "errors"

var (
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionPlanNotFound  = errors.New("subscription plan not found")
)
