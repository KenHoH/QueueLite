package inbound

import (
	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	"QueueLite/internal/queue/app"
	"QueueLite/internal/queue/domain"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type QueueHandlerImpl struct {
	s *app.QueueService
}

func NewQueueHandler(s *app.QueueService) *QueueHandlerImpl {
	return &QueueHandlerImpl{
		s: s,
	}
}

func (h *QueueHandlerImpl) SSEHandler(w http.ResponseWriter, r *http.Request) {
	// sse header
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok { // this to check if w is also Flusher as well, or can be changed to Flusher
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(2 * time.Second) // create a new ticker that triggers every 2 seconds
	defer ticker.Stop()                       // to stop the ticker if the sse handler finished

	// select if like switch to check whos channel is faster that receives data
	select {
	case <-r.Context().Done():
		// Client disconnected
		return

	// every ticks -> triggers.
	case <-ticker.C:
		pubsub := h.s.SubscribeQueueChannel()
		// Always close the PubSub client to prevent connection leaks
		defer pubsub.Close()

		// 2. Start a background goroutine to consume messages
		go func() {
			// pubsub.Channel() returns a Go channel (*redis.Message)
			ch := pubsub.Channel()

			fmt.Println("Waiting for messages...")
			for msg := range ch {
				fmt.Printf("Received message from channel '%s': %s\n", msg.Channel, msg.Payload)
			}
		}()
		// immediately sends buffer data
		flusher.Flush()
	}
}

// TODO: implement
func (h *QueueHandlerImpl) RegisterQueueByQr(w http.ResponseWriter, r *http.Request) {
	// check from context if the userId exist
	// if exist then pass it to normal RegisterQueue function
	// else create a temp user with redis
}

func (h *QueueHandlerImpl) RegisterQueue(w http.ResponseWriter, r *http.Request) {
	var request CreateQueueRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	businessID, ok := parseUUIDValue(w, request.BusinessID, "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}
	var userID *uuid.UUID
	if strings.TrimSpace(request.UserID) != "" {
		parsed, ok := parseUUIDValue(w, request.UserID, "INVALID_USER_ID", "invalid user id")
		if !ok {
			return
		}
		userID = &parsed
	} else {
		generated := uuid.New()
		userID = &generated
	}

	queue, err := h.s.RegisterQueue(r.Context(), domain.Queue{
		BusinessID: businessID,
		UserID:     userID,
		Name:       request.Name,
		Priority:   request.Priority,
	})
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusCreated, NewQueueResponse(queue))
}

func (h *QueueHandlerImpl) GetQueue(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	queue, err := h.s.GetQueue(r.Context(), queueID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, NewQueueResponse(queue))
}

func (h *QueueHandlerImpl) GetQueueState(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	state, err := h.s.GetQueueState(r.Context(), queueID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, QueueStateResponse{State: string(state)})
}

func (h *QueueHandlerImpl) GetAllQueueByBusiness(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}

	queues, err := h.s.GetAllQueueByBusiness(r.Context(), businessID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, NewQueueResponses(queues))
}

func (h *QueueHandlerImpl) GetBusinessPublicQueueSummary(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}

	summary, err := h.s.GetBusinessPublicQueueSummary(r.Context(), businessID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, NewPublicQueueSummaryResponse(summary))
}

func (h *QueueHandlerImpl) GetAllQueueByBusinessFilterState(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}
	state, ok := parseQueueState(w, chi.URLParam(r, "state"))
	if !ok {
		return
	}

	queues, err := h.s.GetAllQueueByBusinessFilterState(r.Context(), businessID, state)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, NewQueueResponses(queues))
}

func (h *QueueHandlerImpl) UpdateQueue(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	var request UpdateQueueRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	if request.Name == nil && request.State == nil && request.Priority == nil {
		writeInvalid(w, "NO_QUEUE_FIELDS", "no queue fields provided")
		return
	}

	queue, err := h.s.GetQueue(r.Context(), queueID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	if request.Name != nil {
		queue.Name = strings.TrimSpace(*request.Name)
	}
	if request.State != nil {
		state, ok := parseQueueState(w, *request.State)
		if !ok {
			return
		}
		queue.State = state
	}
	if request.Priority != nil {
		queue.Priority = *request.Priority
	}

	if err := h.s.UpdateQueue(r.Context(), *queue); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "queue updated"})
}

func (h *QueueHandlerImpl) UpdateState(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	var request UpdateQueueStateRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	state, ok := parseQueueState(w, request.State)
	if !ok {
		return
	}

	if err := h.s.UpdateState(r.Context(), queueID, state); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "queue state updated"})
}

func (h *QueueHandlerImpl) MarkAsDone(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	if err := h.s.MarkAsDone(r.Context(), queueID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "queue marked as done"})
}

func (h *QueueHandlerImpl) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	if err := h.s.DeleteQueue(r.Context(), queueID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "queue deleted"})
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, name string, code string, message string) (uuid.UUID, bool) {
	return parseUUIDValue(w, chi.URLParam(r, name), code, message)
}

func parseUUIDValue(w http.ResponseWriter, value string, code string, message string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		writeInvalid(w, code, message)
		return uuid.Nil, false
	}
	return id, true
}

func parseQueueState(w http.ResponseWriter, value string) (domain.QueueState, bool) {
	state := domain.QueueState(strings.TrimSpace(value))
	switch state {
	case domain.QueueStateWaiting,
		domain.QueueStateCalled,
		domain.QueueStateProcessing,
		domain.QueueStateCancelled,
		domain.QueueStateSkipped,
		domain.QueueStateCompleted:
		return state, true
	default:
		writeInvalid(w, "INVALID_QUEUE_STATE", "invalid queue state")
		return "", false
	}
}

func writeInvalid(w http.ResponseWriter, code string, message string) {
	httpadapter.WriteError(w, apperror.New(
		apperror.KindInvalid,
		code,
		message,
	))
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
