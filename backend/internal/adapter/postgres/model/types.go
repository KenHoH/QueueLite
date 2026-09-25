package model

type BusinessPlanType string

const (
	BusinessPlanTypeFree BusinessPlanType = "free"
	BusinessPlanTypePlus BusinessPlanType = "plus"
	BusinessPlanTypePro  BusinessPlanType = "pro"
	BusinessPlanTypeMax  BusinessPlanType = "max"
)

func IsValidBusinessPlanType(value BusinessPlanType) bool {
	switch value {
	case BusinessPlanTypeFree, BusinessPlanTypePlus, BusinessPlanTypePro, BusinessPlanTypeMax:
		return true
	default:
		return false
	}
}

type UserPlanType string

const (
	UserPlanTypeStandard UserPlanType = "standard"
	UserPlanTypePremium  UserPlanType = "premium"
)

func IsValidUserPlanType(value UserPlanType) bool {
	switch value {
	case UserPlanTypeStandard, UserPlanTypePremium:
		return true
	default:
		return false
	}
}

type BusinessRole string

const (
	BusinessRoleOwner   BusinessRole = "owner"
	BusinessRoleAdmin   BusinessRole = "admin"
	BusinessRoleCounter BusinessRole = "counter"
)

type QueueState string

const (
	QueueStateWaiting    QueueState = "waiting"
	QueueStateCalled     QueueState = "called"
	QueueStateProcessing QueueState = "processing"
	QueueStateCancelled  QueueState = "cancelled"
	QueueStateSkipped    QueueState = "skipped"
	QueueStateCompleted  QueueState = "done"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
)

type SubscriptionType string

const (
	SubscriptionTypeBusiness SubscriptionType = "business"
	SubscriptionTypeUser     SubscriptionType = "user"
)
