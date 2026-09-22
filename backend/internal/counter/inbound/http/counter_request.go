package inbound

type CreateCounterRequest struct {
	BusinessID        string  `json:"businessId"`
	Name              string  `json:"name"`
	CurrentEmployeeID *string `json:"currentEmployeeId,omitempty"`
	CurrentQueueID    *string `json:"currentQueueId,omitempty"`
}

type UpdateCounterRequest struct {
	Name              *string `json:"name"`
	CurrentEmployeeID *string `json:"currentEmployeeId"`
	CurrentQueueID    *string `json:"currentQueueId"`
}
