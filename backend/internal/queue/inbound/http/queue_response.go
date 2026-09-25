package inbound

import "QueueLite/internal/queue/domain"

type QueueResponse struct {
	ID                string  `json:"id"`
	BusinessID        string  `json:"businessId"`
	UserID            *string `json:"userId,omitempty"`
	CalledByCounterID *string `json:"calledByCounterId,omitempty"`
	Name              string  `json:"name"`
	State             string  `json:"state"`
	Priority          bool    `json:"priority"`
}

type QueueStateResponse struct {
	State string `json:"state"`
}

type PublicQueueSummaryResponse struct {
	CurrentQueueName *string `json:"currentQueueName"`
	TotalWaiting     int64   `json:"totalWaiting"`
	NextQueueName    *string `json:"nextQueueName"`
}

func NewQueueResponse(queue *domain.Queue) QueueResponse {
	var userID *string
	if queue.UserID != nil {
		id := queue.UserID.String()
		userID = &id
	}
	var calledByCounterID *string
	if queue.CalledByCounterID != nil {
		id := queue.CalledByCounterID.String()
		calledByCounterID = &id
	}

	return QueueResponse{
		ID:                queue.ID.String(),
		BusinessID:        queue.BusinessID.String(),
		UserID:            userID,
		CalledByCounterID: calledByCounterID,
		Name:              queue.Name,
		State:             string(queue.State),
		Priority:          queue.Priority,
	}
}

func NewQueueResponses(queues []domain.Queue) []QueueResponse {
	responses := make([]QueueResponse, 0, len(queues))
	for i := range queues {
		responses = append(responses, NewQueueResponse(&queues[i]))
	}
	return responses
}

func NewPublicQueueSummaryResponse(summary *domain.PublicQueueSummary) PublicQueueSummaryResponse {
	return PublicQueueSummaryResponse{
		CurrentQueueName: summary.CurrentQueueName,
		TotalWaiting:     summary.TotalWaiting,
		NextQueueName:    summary.NextQueueName,
	}
}
