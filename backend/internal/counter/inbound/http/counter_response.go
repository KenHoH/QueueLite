package inbound

import "QueueLite/internal/counter/domain"

type CounterResponse struct {
	ID                string  `json:"id"`
	BusinessID        string  `json:"businessId"`
	Name              string  `json:"name"`
	CurrentEmployeeID *string `json:"currentEmployeeId,omitempty"`
	CurrentQueueID    *string `json:"currentQueueId,omitempty"`
}

func NewCounterResponse(counter *domain.Counter) CounterResponse {
	var currentEmployeeID *string
	if counter.CurrentEmployeeID != nil {
		id := counter.CurrentEmployeeID.String()
		currentEmployeeID = &id
	}

	var currentQueueID *string
	if counter.CurrentQueueID != nil {
		id := counter.CurrentQueueID.String()
		currentQueueID = &id
	}

	return CounterResponse{
		ID:                counter.ID.String(),
		BusinessID:        counter.BusinessID.String(),
		Name:              counter.Name,
		CurrentEmployeeID: currentEmployeeID,
		CurrentQueueID:    currentQueueID,
	}
}
