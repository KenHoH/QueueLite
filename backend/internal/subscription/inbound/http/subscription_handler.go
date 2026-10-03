package inbound

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/httputil"
	"QueueLite/internal/subscription/app"
	"QueueLite/internal/subscription/domain"

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
	subscriptionID, ok := httputil.ParseUUIDParam(w, r, "subscriptionID", "INVALID_SUBSCRIPTION_ID", "invalid subscription id")
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
	subscriptionID, ok := httputil.ParseUUIDParam(w, r, "subscriptionID", "INVALID_SUBSCRIPTION_ID", "invalid subscription id")
	if !ok {
		return
	}
	var request UpdateSubscriptionRequest
	if err := httputil.DecodeJSON(r, &request); err != nil {
		httputil.WriteInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	var businessPlanID *uuid.UUID
	if strings.TrimSpace(request.BusinessPlanID) != "" {
		parsed, ok := httputil.ParseUUIDValue(w, request.BusinessPlanID, "INVALID_BUSINESS_PLAN_ID", "invalid business plan id")
		if !ok {
			return
		}
		businessPlanID = &parsed
	}
	var userPlanID *uuid.UUID
	if strings.TrimSpace(request.UserPlanID) != "" {
		parsed, ok := httputil.ParseUUIDValue(w, request.UserPlanID, "INVALID_USER_PLAN_ID", "invalid user plan id")
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
	subscriptionID, ok := httputil.ParseUUIDParam(w, r, "subscriptionID", "INVALID_SUBSCRIPTION_ID", "invalid subscription id")
	if !ok {
		return
	}
	var request UpdateSubscriptionTimeRequest
	if err := httputil.DecodeJSON(r, &request); err != nil {
		httputil.WriteInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	startDate, ok := httputil.ParseDateTime(w, request.StartDate, "INVALID_START_DATE", "invalid start date")
	if !ok {
		return
	}
	endDate, ok := httputil.ParseOptionalDateTime(w, request.EndDate, "INVALID_END_DATE", "invalid end date")
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
	subscriptionID, ok := httputil.ParseUUIDParam(w, r, "subscriptionID", "INVALID_SUBSCRIPTION_ID", "invalid subscription id")
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
	businessID, ok := httputil.ParseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
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
	userID, ok := httputil.ParseUUIDParam(w, r, "userID", "INVALID_USER_ID", "invalid user id")
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
	userID, ok := httputil.ParseUUIDParam(w, r, "userID", "INVALID_USER_ID", "invalid user id")
	if !ok {
		return
	}
	var request UseUserSubscriptionRequest
	if err := httputil.DecodeJSON(r, &request); err != nil {
		httputil.WriteInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	businessID, ok := httputil.ParseUUIDValue(w, request.BusinessID, "INVALID_BUSINESS_ID", "invalid business id")
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
	userID, ok := httputil.ParseUUIDParam(w, r, "userID", "INVALID_USER_ID", "invalid user id")
	if !ok {
		return
	}
	var request AddUserSlotRequest
	if err := httputil.DecodeJSON(r, &request); err != nil {
		httputil.WriteInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	if err := h.s.AddUserSlot(r.Context(), userID, request.Amount); err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "user slot added"})
}

func (h *SubscriptionHandlerImpl) DecreaseBusinessCapacity(w http.ResponseWriter, r *http.Request) {
	businessID, ok := httputil.ParseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
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
	subscriptionID, ok := httputil.ParseUUIDParam(w, r, "subscriptionID", "INVALID_SUBSCRIPTION_ID", "invalid subscription id")
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
