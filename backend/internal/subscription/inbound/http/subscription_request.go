package inbound

type CreateSubscriptionRequest struct {
	BusinessID         string  `json:"businessId"`
	SubscriptionPlanID string  `json:"subscriptionPlanId"`
	Type               string  `json:"type"`
	StartDate          string  `json:"startDate"`
	EndDate            *string `json:"endDate"`
	Status             string  `json:"status"`
}

type UpdateSubscriptionRequest struct {
	SubscriptionPlanID string `json:"subscriptionPlanId"`
}

type UpdateSubscriptionTimeRequest struct {
	StartDate string  `json:"startDate"`
	EndDate   *string `json:"endDate"`
}

type CreateSubscriptionPlanRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateSubscriptionPlanRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}
