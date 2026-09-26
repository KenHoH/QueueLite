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
	var businessPlanID *uuid.UUID
	if strings.TrimSpace(request.BusinessPlanID) != "" {
		parsed, ok := parseUUIDValue(w, request.BusinessPlanID, "INVALID_BUSINESS_PLAN_ID", "invalid business plan id")
		if !ok {
			return
		}
		businessPlanID = &parsed
	}
	var userPlanID *uuid.UUID
	if strings.TrimSpace(request.UserPlanID) != "" {
		parsed, ok := parseUUIDValue(w, request.UserPlanID, "INVALID_USER_PLAN_ID", "invalid user plan id")
		if !ok {
			return
		}
		userPlanID = &parsed
	}
	if err := h.s.UpdateSubscription(r.Context(), subscriptionID, businessPlanID, userPlanID); err != nil {
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

func (h *SubscriptionHandlerImpl) GetBusinessSubscriptionInfo(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}
	info, err := h.s.GetBusinessSubscriptionInfo(r.Context(), businessID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, NewBusinessSubscriptionInfoResponse(info))
}

func (h *SubscriptionHandlerImpl) GetUserSubscriptionInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUUIDParam(w, r, "userID", "INVALID_USER_ID", "invalid user id")
	if !ok {
		return
	}
	info, err := h.s.GetUserSubscriptionInfo(r.Context(), userID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, NewUserSubscriptionInfoResponse(info))
}

func (h *SubscriptionHandlerImpl) UseUserSubscription(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUUIDParam(w, r, "userID", "INVALID_USER_ID", "invalid user id")
	if !ok {
		return
	}
	var request UseUserSubscriptionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	businessID, ok := parseUUIDValue(w, request.BusinessID, "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}
	result, err := h.s.UseUserSubscription(r.Context(), userID, businessID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, NewUseUserSubscriptionResponse(result))
}

func (h *SubscriptionHandlerImpl) AddUserSlot(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUUIDParam(w, r, "userID", "INVALID_USER_ID", "invalid user id")
	if !ok {
		return
	}
	var request AddUserSlotRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	if err := h.s.AddUserSlot(r.Context(), userID, request.Amount); err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "user slot added"})
}

func (h *SubscriptionHandlerImpl) DecreaseBusinessCapacity(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}
	if err := h.s.DecreaseBusinessCapacity(r.Context(), businessID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "business capacity decreased"})
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
