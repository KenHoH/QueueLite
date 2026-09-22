package inbound

import (
	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	"QueueLite/internal/queue/app"
	"QueueLite/internal/queue/domain"
	"encoding/json"
	"net/http"
	"strings"

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

func (h *QueueHandlerImpl) Routes() http.Handler {
	router := chi.NewRouter()

	router.Post("/", h.RegisterQueue)
	router.Get("/business/{businessID}", h.GetAllQueueByBusiness)
	router.Get("/business/{businessID}/state/{state}", h.GetAllQueueByBusinessFilterState)
	router.Get("/{queueID}", h.GetQueue)
	router.Get("/{queueID}/state", h.GetQueueState)
	router.Put("/{queueID}", h.UpdateQueue)
	router.Patch("/{queueID}/state", h.UpdateState)
	router.Patch("/{queueID}/done", h.MarkAsDone)
	router.Patch("/{queueID}/counter/{counterID}", h.AssignToCounter)
	router.Delete("/{queueID}/counter/{counterID}", h.RemoveFromCounter)
	router.Delete("/{queueID}", h.DeleteQueue)

	return router
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
	userID, ok := parseUUIDValue(w, request.UserID, "INVALID_USER_ID", "invalid user id")
	if !ok {
		return
	}

	var counterID *uuid.UUID
	if request.CounterID != nil && strings.TrimSpace(*request.CounterID) != "" {
		parsed, ok := parseUUIDValue(w, *request.CounterID, "INVALID_COUNTER_ID", "invalid counter id")
		if !ok {
			return
		}
		counterID = &parsed
	}

	if request.Name == "" {
		writeInvalid(w, "QUEUE_NAME_REQUIRED", "Missing queue name")
		return
	}

	queue, err := h.s.RegisterQueue(r.Context(), domain.Queue{
		BusinessID: businessID,
		UserID:     userID,
		CounterID:  counterID,
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
	if request.Name == nil && request.CounterID == nil && request.State == nil && request.Priority == nil {
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
	if request.CounterID != nil {
		if strings.TrimSpace(*request.CounterID) == "" {
			queue.CounterID = nil
		} else {
			counterID, ok := parseUUIDValue(w, *request.CounterID, "INVALID_COUNTER_ID", "invalid counter id")
			if !ok {
				return
			}
			queue.CounterID = &counterID
		}
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

func (h *QueueHandlerImpl) AssignToCounter(w http.ResponseWriter, r *http.Request) {
	queueID, counterID, ok := h.parseQueueCounterParams(w, r)
	if !ok {
		return
	}

	if err := h.s.AssignToCounter(r.Context(), queueID, counterID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "queue assigned to counter"})
}

func (h *QueueHandlerImpl) RemoveFromCounter(w http.ResponseWriter, r *http.Request) {
	queueID, counterID, ok := h.parseQueueCounterParams(w, r)
	if !ok {
		return
	}

	if err := h.s.RemoveFromCounter(r.Context(), queueID, counterID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "queue removed from counter"})
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

func (h *QueueHandlerImpl) parseQueueCounterParams(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	counterID, ok := parseUUIDParam(w, r, "counterID", "INVALID_COUNTER_ID", "invalid counter id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return queueID, counterID, true
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
		domain.QueueStateProcess,
		domain.QueueStateCanceled,
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
