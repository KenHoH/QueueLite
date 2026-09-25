package app

import "QueueLite/internal/subscription/domain"

func IsValidSubscriptionType(value domain.SubscriptionType) bool {
	switch value {
	case domain.SubscriptionTypeBusiness, domain.SubscriptionTypeUser:
		return true
	default:
		return false
	}
}

func IsValidSubscriptionStatus(value domain.SubscriptionStatus) bool {
	switch value {
	case domain.SubscriptionStatusActive, domain.SubscriptionStatusInactive:
		return true
	default:
		return false
	}
}

func IsValidBusinessPlanType(value domain.BusinessPlanType) bool {
	switch value {
	case domain.BusinessPlanTypeFree, domain.BusinessPlanTypePlus, domain.BusinessPlanTypePro, domain.BusinessPlanTypeMax:
		return true
	default:
		return false
	}
}

func IsValidUserPlanType(value domain.UserPlanType) bool {
	switch value {
	case domain.UserPlanTypeStandard, domain.UserPlanTypePremium:
		return true
	default:
		return false
	}
}
