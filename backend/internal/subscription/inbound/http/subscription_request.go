package inbound

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
