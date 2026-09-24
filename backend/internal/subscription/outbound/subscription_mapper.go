package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/subscription/domain"
)

func toSubscriptionRecord(subscription *domain.Subscription) model.Subscription {
	return model.Subscription{
		ID:             subscription.ID,
		BusinessID:     subscription.BusinessID,
		UserID:         subscription.UserID,
		BusinessPlanID: subscription.BusinessPlanID,
		UserPlanID:     subscription.UserPlanID,
		Type:           model.SubscriptionType(subscription.Type),
		StartDate:      subscription.StartDate,
		EndDate:        subscription.EndDate,
		Status:         model.SubscriptionStatus(subscription.Status),
	}
}

func toDomainSubscription(record *model.Subscription) *domain.Subscription {
	return &domain.Subscription{
		ID:             record.ID,
		BusinessID:     record.BusinessID,
		UserID:         record.UserID,
		BusinessPlanID: record.BusinessPlanID,
		UserPlanID:     record.UserPlanID,
		Type:           domain.SubscriptionType(record.Type),
		StartDate:      record.StartDate,
		EndDate:        record.EndDate,
		Status:         domain.SubscriptionStatus(record.Status),
		CreatedAt:      record.CreatedAt,
	}
}

func toDomainSubscriptions(records []model.Subscription) []domain.Subscription {
	subscriptions := make([]domain.Subscription, 0, len(records))
	for i := range records {
		subscriptions = append(subscriptions, *toDomainSubscription(&records[i]))
	}
	return subscriptions
}

func toBusinessPlanRecord(plan *domain.BusinessPlan) model.BusinessPlan {
	return model.BusinessPlan{
		ID:                  plan.ID,
		BusinessPlanType:    model.BusinessPlanType(plan.BusinessPlanType),
		Description:         plan.Description,
		Price:               plan.Price,
		Capacity:            plan.Capacity,
		Analysis:            plan.Analysis,
		Insight:             plan.Insight,
		PrioritySupport:     plan.PrioritySupport,
		LastCapacityResetAt: plan.LastCapacityResetAt,
	}
}

func toDomainBusinessPlan(record *model.BusinessPlan) *domain.BusinessPlan {
	return &domain.BusinessPlan{
		ID:                  record.ID,
		BusinessPlanType:    domain.BusinessPlanType(record.BusinessPlanType),
		Description:         record.Description,
		Price:               record.Price,
		Capacity:            record.Capacity,
		Analysis:            record.Analysis,
		Insight:             record.Insight,
		PrioritySupport:     record.PrioritySupport,
		LastCapacityResetAt: record.LastCapacityResetAt,
		CreatedAt:           record.CreatedAt,
		UpdatedAt:           record.UpdatedAt,
	}
}

func toUserPlanRecord(plan *domain.UserPlan) model.UserPlan {
	return model.UserPlan{
		ID:               plan.ID,
		UserPlanType:     model.UserPlanType(plan.UserPlanType),
		Name:             plan.Name,
		Description:      plan.Description,
		Price:            plan.Price,
		Slots:            plan.Slots,
		LastSlotsResetAt: plan.LastSlotsResetAt,
	}
}

func toDomainUserPlan(record *model.UserPlan) *domain.UserPlan {
	return &domain.UserPlan{
		ID:               record.ID,
		UserPlanType:     domain.UserPlanType(record.UserPlanType),
		Name:             record.Name,
		Description:      record.Description,
		Price:            record.Price,
		Slots:            record.Slots,
		LastSlotsResetAt: record.LastSlotsResetAt,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}
