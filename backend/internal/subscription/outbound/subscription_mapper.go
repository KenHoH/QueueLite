package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/subscription/domain"
)

func toSubscriptionRecord(subscription *domain.Subscription) model.Subscription {
	return model.Subscription{
		ID:                 subscription.ID,
		BusinessID:         subscription.BusinessID,
		SubscriptionPlanID: subscription.SubscriptionPlanID,
		Type:               model.SubscriptionType(subscription.Type),
		StartDate:          subscription.StartDate,
		EndDate:            subscription.EndDate,
		Status:             model.SubscriptionStatus(subscription.Status),
	}
}

func toDomainSubscription(record *model.Subscription) *domain.Subscription {
	return &domain.Subscription{
		ID:                 record.ID,
		BusinessID:         record.BusinessID,
		SubscriptionPlanID: record.SubscriptionPlanID,
		Type:               domain.SubscriptionType(record.Type),
		StartDate:          record.StartDate,
		EndDate:            record.EndDate,
		Status:             domain.SubscriptionStatus(record.Status),
		CreatedAt:          record.CreatedAt,
	}
}

func toDomainSubscriptions(records []model.Subscription) []domain.Subscription {
	subscriptions := make([]domain.Subscription, 0, len(records))
	for i := range records {
		subscriptions = append(subscriptions, *toDomainSubscription(&records[i]))
	}
	return subscriptions
}

func toSubscriptionPlanRecord(plan *domain.SubscriptionPlan) model.SubscriptionPlan {
	return model.SubscriptionPlan{
		ID:          plan.ID,
		Name:        plan.Name,
		Description: plan.Description,
	}
}

func toDomainSubscriptionPlan(record *model.SubscriptionPlan) *domain.SubscriptionPlan {
	return &domain.SubscriptionPlan{
		ID:          record.ID,
		Name:        record.Name,
		Description: record.Description,
	}
}
