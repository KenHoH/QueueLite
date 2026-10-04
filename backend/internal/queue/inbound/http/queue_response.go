package inbound

import (
	"time"

	"QueueLite/internal/queue/app"
	"QueueLite/internal/queue/domain"
)

type QueueResponse struct {
	ID                string     `json:"id"`
	BusinessID        string     `json:"businessId"`
	UserID            *string    `json:"userId,omitempty"`
	CalledByCounterID *string    `json:"calledByCounterId,omitempty"`
	Name              string     `json:"name"`
	State             string     `json:"state"`
	Priority          bool       `json:"priority"`
	CalledAt          *time.Time `json:"calledAt,omitempty"`
	ProcessingAt      *time.Time `json:"processingAt,omitempty"`
	DoneAt            *time.Time `json:"doneAt,omitempty"`
	CancelledAt       *time.Time `json:"cancelledAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type QueueStateResponse struct {
	State string `json:"state"`
}

type QueueQRResolveResponse struct {
	BusinessID        string   `json:"businessId"`
	Authenticated     bool     `json:"authenticated"`
	RequiresGuestForm bool     `json:"requiresGuestForm"`
	RequiredFields    []string `json:"requiredFields,omitempty"`
}

type PublicQueueSummaryResponse struct {
	CurrentQueueName *string `json:"currentQueueName"`
	TotalWaiting     int64   `json:"totalWaiting"`
	NextQueueName    *string `json:"nextQueueName"`
}

type CustomerStatusBusinessResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Operational bool   `json:"operational"`
}

type CustomerCounterStatusResponse struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	CurrentQueueID   *string `json:"currentQueueId,omitempty"`
	CurrentQueueName *string `json:"currentQueueName,omitempty"`
	State            string  `json:"state"`
}

type CurrentlyServingResponse struct {
	CounterID   string `json:"counterId"`
	CounterName string `json:"counterName"`
	QueueID     string `json:"queueId"`
	QueueName   string `json:"queueName"`
	State       string `json:"state"`
}

type QueuePositionResponse struct {
	Position  int    `json:"position"`
	Ahead     int    `json:"ahead"`
	QueueName string `json:"queueName"`
}

type QueueSnapshotResponse struct {
	QueueID   string `json:"queueId"`
	QueueName string `json:"queueName"`
}

type CustomerQueueStatusResponse struct {
	Queue                QueueResponse                   `json:"queue"`
	Business             CustomerStatusBusinessResponse  `json:"business"`
	Counters             []CustomerCounterStatusResponse `json:"counters"`
	TotalCounters        int                             `json:"totalCounters"`
	ActiveCounters       int                             `json:"activeCounters"`
	CurrentlyServing     []CurrentlyServingResponse      `json:"currentlyServing"`
	NextQueue            *QueueSnapshotResponse          `json:"nextQueue,omitempty"`
	CustomerPosition     *QueuePositionResponse          `json:"customerPosition,omitempty"`
	TotalWaiting         int                             `json:"totalWaiting"`
	TotalActiveQueues    int                             `json:"totalActiveQueues"`
	EstimatedWaitMinutes *int                            `json:"estimatedWaitMinutes,omitempty"`
	UpdatedAt            time.Time                       `json:"updatedAt"`
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
		CalledAt:          queue.CalledAt,
		ProcessingAt:      queue.ProcessingAt,
		DoneAt:            queue.DoneAt,
		CancelledAt:       queue.CancelledAt,
		CreatedAt:         queue.CreatedAt,
		UpdatedAt:         queue.UpdatedAt,
	}
}

func NewQueueSnapshotResponses(items []app.QueueSnapshotItem) []QueueSnapshotResponse {
	responses := make([]QueueSnapshotResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, QueueSnapshotResponse{
			QueueID:   item.QueueID,
			QueueName: item.QueueName,
		})
	}
	return responses
}

func NewQueueResponses(queues []domain.Queue) []QueueResponse {
	responses := make([]QueueResponse, 0, len(queues))
	for i := range queues {
		responses = append(responses, NewQueueResponse(&queues[i]))
	}
	return responses
}

func NewCustomerQueueStatusResponse(status *app.CustomerQueueStatus) CustomerQueueStatusResponse {
	counters := make([]CustomerCounterStatusResponse, 0, len(status.Counters))
	for _, counter := range status.Counters {
		var queueID *string
		if counter.CurrentQueueID != nil {
			value := counter.CurrentQueueID.String()
			queueID = &value
		}
		counters = append(counters, CustomerCounterStatusResponse{ID: counter.ID.String(), Name: counter.Name, CurrentQueueID: queueID, CurrentQueueName: counter.CurrentQueueName, State: counter.State})
	}
	serving := make([]CurrentlyServingResponse, 0, len(status.CurrentlyServing))
	for _, item := range status.CurrentlyServing {
		serving = append(serving, CurrentlyServingResponse{CounterID: item.CounterID.String(), CounterName: item.CounterName, QueueID: item.QueueID.String(), QueueName: item.QueueName, State: string(item.State)})
	}
	var next *QueueSnapshotResponse
	if status.NextQueue != nil {
		next = &QueueSnapshotResponse{QueueID: status.NextQueue.QueueID, QueueName: status.NextQueue.QueueName}
	}
	var position *QueuePositionResponse
	if status.CustomerPosition != nil {
		position = &QueuePositionResponse{Position: status.CustomerPosition.Position, Ahead: status.CustomerPosition.Ahead, QueueName: status.CustomerPosition.QueueName}
	}
	return CustomerQueueStatusResponse{
		Queue:    NewQueueResponse(&status.Queue),
		Business: CustomerStatusBusinessResponse{ID: status.Business.ID.String(), Name: status.Business.Name, Operational: status.Business.Operational},
		Counters: counters, TotalCounters: status.TotalCounters, ActiveCounters: status.ActiveCounters,
		CurrentlyServing: serving, NextQueue: next, CustomerPosition: position,
		TotalWaiting: status.TotalWaiting, TotalActiveQueues: status.TotalActiveQueues,
		EstimatedWaitMinutes: status.EstimatedWaitMinutes, UpdatedAt: status.UpdatedAt,
	}
}

func NewPublicQueueSummaryResponse(summary *domain.PublicQueueSummary) PublicQueueSummaryResponse {
	return PublicQueueSummaryResponse{
		CurrentQueueName: summary.CurrentQueueName,
		TotalWaiting:     summary.TotalWaiting,
		NextQueueName:    summary.NextQueueName,
	}
}
