package inbound

import (
	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	"QueueLite/internal/counter/app"
	"QueueLite/internal/counter/domain"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CounterHandlerImpl struct {
	s *app.CounterService
}

func NewCounterHandler(s *app.CounterService) *CounterHandlerImpl {
	return &CounterHandlerImpl{s: s}
}

func (h *CounterHandlerImpl) PublicRoutes() http.Handler {
	router := chi.NewRouter()

	router.Get("/{counterID}", h.GetCounter)
	return router
}

func (h *CounterHandlerImpl) PrivateRoutes() http.Handler {
	router := chi.NewRouter()

	router.Post("/", h.CreateCounter)
	router.Post("/{counterID}/business/{businessID}/call-next", h.CallNextQueue)
	router.Post("/{counterID}/queues/{queueID}/process", h.ProcessCalledQueue)
	router.Post("/{counterID}/queues/{queueID}/skip", h.SkipQueue)
	router.Put("/{counterID}", h.UpdateCounter)
	router.Delete("/{counterID}", h.DeleteCounter)
	return router
}

func (h *CounterHandlerImpl) CreateCounter(w http.ResponseWriter, r *http.Request) {
	var request CreateCounterRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	businessID, ok := parseUUIDValue(w, request.BusinessID, "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}
	employeeID, ok := parseOptionalUUIDValue(w, request.CurrentEmployeeID, "INVALID_EMPLOYEE_ID", "invalid employee id")
	if !ok {
		return
	}
	queueID, ok := parseOptionalUUIDValue(w, request.CurrentQueueID, "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	counter, err := h.s.CreateCounter(r.Context(), domain.Counter{
		BusinessID:        businessID,
		Name:              request.Name,
		CurrentEmployeeID: employeeID,
		CurrentQueueID:    queueID,
	})
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusCreated, NewCounterResponse(counter))
}

func (h *CounterHandlerImpl) GetCounter(w http.ResponseWriter, r *http.Request) {
	counterID, ok := parseUUIDParam(w, r, "counterID", "INVALID_COUNTER_ID", "invalid counter id")
	if !ok {
		return
	}

	counter, err := h.s.GetCounter(r.Context(), counterID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, NewCounterResponse(counter))
}

func (h *CounterHandlerImpl) UpdateCounter(w http.ResponseWriter, r *http.Request) {
	counterID, ok := parseUUIDParam(w, r, "counterID", "INVALID_COUNTER_ID", "invalid counter id")
	if !ok {
		return
	}

	var request UpdateCounterRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	if request.Name == nil && request.CurrentEmployeeID == nil && request.CurrentQueueID == nil {
		writeInvalid(w, "NO_COUNTER_FIELDS", "no counter fields provided")
		return
	}

	if request.Name != nil {
		current, err := h.s.GetCounter(r.Context(), counterID)
		if err != nil {
			httpadapter.WriteError(w, err)
			return
		}
		current.Name = *request.Name
		if err := h.s.UpdateCounter(r.Context(), *current); err != nil {
			httpadapter.WriteError(w, err)
			return
		}
	}
	if request.CurrentEmployeeID != nil {
		employeeID, ok := parseOptionalUUIDValue(w, request.CurrentEmployeeID, "INVALID_EMPLOYEE_ID", "invalid employee id")
		if !ok {
			return
		}
		if err := h.s.UpdateCounterEmployee(r.Context(), counterID, employeeID); err != nil {
			httpadapter.WriteError(w, err)
			return
		}
	}
	if request.CurrentQueueID != nil {
		if strings.TrimSpace(*request.CurrentQueueID) == "" {
			if err := h.s.ClearCounterCustomer(r.Context(), counterID); err != nil {
				httpadapter.WriteError(w, err)
				return
			}
		} else {
			queueID, ok := parseUUIDValue(w, *request.CurrentQueueID, "INVALID_QUEUE_ID", "invalid queue id")
			if !ok {
				return
			}
			if err := h.s.UpdateCounterCustomer(r.Context(), counterID, queueID); err != nil {
				httpadapter.WriteError(w, err)
				return
			}
		}
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "counter updated"})
}

func (h *CounterHandlerImpl) CallNextQueue(w http.ResponseWriter, r *http.Request) {
	counterID, ok := parseUUIDParam(w, r, "counterID", "INVALID_COUNTER_ID", "invalid counter id")
	if !ok {
		return
	}
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}
	queue, err := h.s.CallNextQueue(r.Context(), counterID, businessID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	if queue == nil {
		httpadapter.WriteJSON(w, http.StatusOK, map[string]any{"queue": nil})
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"queueId": queue.ID.String(), "queueName": queue.Name})
}

func (h *CounterHandlerImpl) ProcessCalledQueue(w http.ResponseWriter, r *http.Request) {
	counterID, queueID, ok := h.parseCounterQueueParams(w, r)
	if !ok {
		return
	}
	queue, err := h.s.ProcessCalledQueue(r.Context(), counterID, queueID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"queueId": queue.ID.String(), "state": string(queue.State)})
}

func (h *CounterHandlerImpl) SkipQueue(w http.ResponseWriter, r *http.Request) {
	counterID, queueID, ok := h.parseCounterQueueParams(w, r)
	if !ok {
		return
	}
	queue, err := h.s.SkipQueue(r.Context(), counterID, queueID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	if queue == nil {
		httpadapter.WriteJSON(w, http.StatusOK, map[string]any{"queue": nil})
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"queueId": queue.ID.String(), "queueName": queue.Name})
}

func (h *CounterHandlerImpl) parseCounterQueueParams(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	counterID, ok := parseUUIDParam(w, r, "counterID", "INVALID_COUNTER_ID", "invalid counter id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return counterID, queueID, true
}

func (h *CounterHandlerImpl) DeleteCounter(w http.ResponseWriter, r *http.Request) {
	counterID, ok := parseUUIDParam(w, r, "counterID", "INVALID_COUNTER_ID", "invalid counter id")
	if !ok {
		return
	}

	if err := h.s.DeleteCounter(r.Context(), counterID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "counter deleted"})
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

func parseOptionalUUIDValue(w http.ResponseWriter, value *string, code string, message string) (*uuid.UUID, bool) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, true
	}
	id, ok := parseUUIDValue(w, *value, code, message)
	if !ok {
		return nil, false
	}
	return &id, true
}

func writeInvalid(w http.ResponseWriter, code string, message string) {
	httpadapter.WriteError(w, apperror.New(apperror.KindInvalid, code, message))
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
