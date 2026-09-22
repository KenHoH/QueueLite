package inbound

import (
	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	"QueueLite/internal/subscription/app"
	"QueueLite/internal/subscription/domain"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SubscriptionHandlerImpl struct {
	s *app.SubscriptionService
}

func NewSubscriptionHandler(s *app.SubscriptionService) *SubscriptionHandlerImpl {
	return &SubscriptionHandlerImpl{s: s}
}

func (h *SubscriptionHandlerImpl) Routes() http.Handler {
	router := chi.NewRouter()

	router.Post("/plans", h.CreatePlan)
	router.Get("/plans/{planID}", h.GetPlan)
	router.Put("/plans/{planID}", h.UpdatePlan)
	router.Delete("/plans/{planID}", h.DeletePlan)
	router.Get("/", h.GetAllSubscription)
	router.Post("/", h.CreateSubscription)
	router.Get("/{subscriptionID}", h.GetSubscription)
	router.Put("/{subscriptionID}", h.UpdateSubscription)
	router.Patch("/{subscriptionID}/time", h.UpdateSubscriptionTime)
	router.Patch("/{subscriptionID}/activate", h.ActivateUserSubscription)
	router.Patch("/{subscriptionID}/deactivate", h.DeactivateUserSubscription)
	router.Delete("/{subscriptionID}", h.DeleteSubscription)

	return router
}

func (h *SubscriptionHandlerImpl) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var request CreateSubscriptionPlanRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	plan, err := h.s.CreatePlan(r.Context(), domain.SubscriptionPlan{Name: request.Name, Description: request.Description})
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusCreated, NewSubscriptionPlanResponse(plan))
}

func (h *SubscriptionHandlerImpl) GetPlan(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUIDParam(w, r, "planID", "INVALID_PLAN_ID", "invalid plan id")
	if !ok {
		return
	}
	plan, err := h.s.GetPlan(r.Context(), planID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, NewSubscriptionPlanResponse(plan))
}

func (h *SubscriptionHandlerImpl) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUIDParam(w, r, "planID", "INVALID_PLAN_ID", "invalid plan id")
	if !ok {
		return
	}
	var request UpdateSubscriptionPlanRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	if request.Name == nil && request.Description == nil {
		writeInvalid(w, "NO_PLAN_FIELDS", "no subscription plan fields provided")
		return
	}
	if request.Name != nil {
		if err := h.s.UpdatePlanName(r.Context(), planID, *request.Name); err != nil {
			httpadapter.WriteError(w, err)
			return
		}
	}
	if request.Description != nil {
		if err := h.s.UpdatePlanDescription(r.Context(), planID, *request.Description); err != nil {
			httpadapter.WriteError(w, err)
			return
		}
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "subscription plan updated"})
}

func (h *SubscriptionHandlerImpl) DeletePlan(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUIDParam(w, r, "planID", "INVALID_PLAN_ID", "invalid plan id")
	if !ok {
		return
	}
	if err := h.s.DeletePlan(r.Context(), planID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "subscription plan deleted"})
}

func (h *SubscriptionHandlerImpl) GetAllSubscription(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	subscriptions, nextCursor, err := h.s.GetAllSubscription(r.Context(), parseSubscriptionCursor(r), limit)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]any{
		"data":       NewSubscriptionResponses(subscriptions),
		"nextCursor": NewSubscriptionCursorResponse(nextCursor),
	})
}

func (h *SubscriptionHandlerImpl) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	var request CreateSubscriptionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	businessID, ok := parseUUIDValue(w, request.BusinessID, "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}
	planID, ok := parseUUIDValue(w, request.SubscriptionPlanID, "INVALID_PLAN_ID", "invalid plan id")
	if !ok {
		return
	}
	startDate, ok := parseDateTime(w, request.StartDate, "INVALID_START_DATE", "invalid start date")
	if !ok {
		return
	}
	endDate, ok := parseOptionalDateTime(w, request.EndDate, "INVALID_END_DATE", "invalid end date")
	if !ok {
		return
	}

	subscription, err := h.s.CreateSubscription(r.Context(), domain.Subscription{
		BusinessID:         businessID,
		SubscriptionPlanID: planID,
		Type:               domain.SubscriptionType(strings.TrimSpace(request.Type)),
		StartDate:          startDate,
		EndDate:            endDate,
		Status:             domain.SubscriptionStatus(strings.TrimSpace(request.Status)),
	})
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusCreated, NewSubscriptionResponse(subscription))
}

func (h *SubscriptionHandlerImpl) GetSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionID, ok := parseUUIDParam(w, r, "subscriptionID", "INVALID_SUBSCRIPTION_ID", "invalid subscription id")
	if !ok {
		return
	}
	subscription, err := h.s.GetSubscription(r.Context(), subscriptionID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, NewSubscriptionResponse(subscription))
}

func (h *SubscriptionHandlerImpl) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionID, ok := parseUUIDParam(w, r, "subscriptionID", "INVALID_SUBSCRIPTION_ID", "invalid subscription id")
	if !ok {
		return
	}
	var request UpdateSubscriptionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	planID, ok := parseUUIDValue(w, request.SubscriptionPlanID, "INVALID_PLAN_ID", "invalid plan id")
	if !ok {
		return
	}
	if err := h.s.UpdateSubscription(r.Context(), subscriptionID, planID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "subscription updated"})
}

func (h *SubscriptionHandlerImpl) UpdateSubscriptionTime(w http.ResponseWriter, r *http.Request) {
	subscriptionID, ok := parseUUIDParam(w, r, "subscriptionID", "INVALID_SUBSCRIPTION_ID", "invalid subscription id")
	if !ok {
		return
	}
	var request UpdateSubscriptionTimeRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	startDate, ok := parseDateTime(w, request.StartDate, "INVALID_START_DATE", "invalid start date")
	if !ok {
		return
	}
	endDate, ok := parseOptionalDateTime(w, request.EndDate, "INVALID_END_DATE", "invalid end date")
	if !ok {
		return
	}
	if err := h.s.UpdateSubscriptionTime(r.Context(), subscriptionID, startDate, endDate); err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "subscription time updated"})
}

func (h *SubscriptionHandlerImpl) ActivateUserSubscription(w http.ResponseWriter, r *http.Request) {
	h.updateSubscriptionStatus(w, r, true)
}

func (h *SubscriptionHandlerImpl) DeactivateUserSubscription(w http.ResponseWriter, r *http.Request) {
	h.updateSubscriptionStatus(w, r, false)
}

func (h *SubscriptionHandlerImpl) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionID, ok := parseUUIDParam(w, r, "subscriptionID", "INVALID_SUBSCRIPTION_ID", "invalid subscription id")
	if !ok {
		return
	}
	if err := h.s.DeleteSubscription(r.Context(), subscriptionID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "subscription deleted"})
}

func (h *SubscriptionHandlerImpl) updateSubscriptionStatus(w http.ResponseWriter, r *http.Request, active bool) {
	subscriptionID, ok := parseUUIDParam(w, r, "subscriptionID", "INVALID_SUBSCRIPTION_ID", "invalid subscription id")
	if !ok {
		return
	}
	var err error
	if active {
		err = h.s.ActivateUserSubscription(r.Context(), subscriptionID)
	} else {
		err = h.s.DeactivateUserSubscription(r.Context(), subscriptionID)
	}
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	message := "subscription deactivated"
	if active {
		message = "subscription activated"
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": message})
}

func parseSubscriptionCursor(r *http.Request) *domain.SubscriptionCursor {
	createdAtValue := strings.TrimSpace(r.URL.Query().Get("cursorCreatedAt"))
	idValue := strings.TrimSpace(r.URL.Query().Get("cursorID"))
	if createdAtValue == "" || idValue == "" {
		return nil
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtValue)
	if err != nil {
		return nil
	}
	id, err := uuid.Parse(idValue)
	if err != nil {
		return nil
	}
	return &domain.SubscriptionCursor{CreatedAt: createdAt, ID: id}
}

func parseDateTime(w http.ResponseWriter, value string, code string, message string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse("2006-01-02", value)
	}
	if err != nil {
		writeInvalid(w, code, message)
		return time.Time{}, false
	}
	return parsed, true
}

func parseOptionalDateTime(w http.ResponseWriter, value *string, code string, message string) (*time.Time, bool) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, true
	}
	parsed, ok := parseDateTime(w, *value, code, message)
	if !ok {
		return nil, false
	}
	return &parsed, true
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

func writeInvalid(w http.ResponseWriter, code string, message string) {
	httpadapter.WriteError(w, apperror.New(apperror.KindInvalid, code, message))
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
