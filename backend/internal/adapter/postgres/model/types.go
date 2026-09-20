package model

type BusinessRole string

const (
	BusinessRoleOwner    BusinessRole = "owner"
	BusinessRoleAdmin    BusinessRole = "admin"
	BusinessRoleEmployee BusinessRole = "employee"
)

type QueueState string

const (
	QueueStateWaiting   QueueState = "waiting"
	QueueStateCalled    QueueState = "called"
	QueueStateProcess   QueueState = "process"
	QueueStateCanceled  QueueState = "canceled"
	QueueStateCompleted QueueState = "done"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
)

type SubscriptionType string

const (
	SubscriptionTypeMonthly SubscriptionType = "business"
	SubscriptionTypeYearly  SubscriptionType = "person"
)
