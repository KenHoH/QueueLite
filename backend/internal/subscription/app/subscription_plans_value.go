package app

import (
	"QueueLite/internal/subscription/domain"
	"strings"
	"time"
)

func DefaultBusinessPlan(planType domain.BusinessPlanType, now time.Time) domain.BusinessPlan {
	plan := domain.BusinessPlan{BusinessPlanType: planType, LastCapacityResetAt: now}
	switch planType {
	case domain.BusinessPlanTypePlus:
		plan.Description = "Plus business plan"
		plan.Price = 299
		plan.Capacity = 500
		plan.Analysis = true
		plan.PrioritySupport = true
	case domain.BusinessPlanTypePro:
		plan.Description = "Pro business plan"
		plan.Price = 699
		plan.Capacity = 2000
		plan.Analysis = true
		plan.Insight = true
		plan.PrioritySupport = true
	case domain.BusinessPlanTypeMax:
		plan.Description = "Max business plan"
		plan.Price = 899
		plan.Capacity = 10000
		plan.Analysis = true
		plan.Insight = true
		plan.PrioritySupport = true
	default:
		plan.BusinessPlanType = domain.BusinessPlanTypeFree
		plan.Description = "Free business plan"
		plan.Price = 0
		plan.Capacity = 100
		plan.Analysis = true
		plan.PrioritySupport = false
	}
	return plan
}

func DefaultUserPlan(planType domain.UserPlanType, now time.Time) domain.UserPlan {
	plan := domain.UserPlan{UserPlanType: planType, LastSlotsResetAt: now}
	switch planType {
	case domain.UserPlanTypePremium:
		plan.Name = "Premium"
		plan.Description = "Premium user plan"
		plan.Price = 499
		plan.Slots = 4
	default:
		plan.UserPlanType = domain.UserPlanTypeStandard
		plan.Name = "Standard"
		plan.Description = "Standard user plan"
		plan.Price = 0
		plan.Slots = 0
	}
	plan.Name = strings.TrimSpace(plan.Name)
	return plan
}
