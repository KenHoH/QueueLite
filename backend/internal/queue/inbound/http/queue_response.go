package inbound

import "QueueLite/internal/queue/domain"

type QueueResponse struct {
	ID         string  `json:"id"`
	BusinessID string  `json:"businessId"`
	UserID     string  `json:"userId"`
	CounterID  *string `json:"counterId,omitempty"`
	Name       string  `json:"name"`
	State      string  `json:"state"`
	Priority   bool    `json:"priority"`
}

type QueueStateResponse struct {
	State string `json:"state"`
}

func NewQueueResponse(queue *domain.Queue) QueueResponse {
	var counterID *string
	if queue.CounterID != nil {
		id := queue.CounterID.String()
		counterID = &id
	}

	return QueueResponse{
		ID:         queue.ID.String(),
		BusinessID: queue.BusinessID.String(),
		UserID:     queue.UserID.String(),
		CounterID:  counterID,
		Name:       queue.Name,
		State:      string(queue.State),
		Priority:   queue.Priority,
	}
}

func NewQueueResponses(queues []domain.Queue) []QueueResponse {
	responses := make([]QueueResponse, 0, len(queues))
	for i := range queues {
		responses = append(responses, NewQueueResponse(&queues[i]))
	}
	return responses
}
