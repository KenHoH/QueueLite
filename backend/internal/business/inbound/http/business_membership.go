package inbound

import (
	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/middleware"
	"github.com/google/uuid"
	"net/http"
)

type BusinessMembershipResponse struct {
	BusinessResponse
	Role string `json:"role"`
}

func currentUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	value, ok := middleware.UserIdFromContext(r.Context())
	id, err := uuid.Parse(value)
	if !ok || err != nil || id == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return uuid.Nil, false
	}
	return id, true
}

func (h *BusinessHandlerImpl) GetMyBusinesses(w http.ResponseWriter, r *http.Request) {
	id, ok := currentUserID(w, r)
	if !ok {
		return
	}
	items, err := h.s.GetUserBusinesses(r.Context(), id)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	response := make([]BusinessMembershipResponse, 0, len(items))
	for _, item := range items {
		response = append(response, BusinessMembershipResponse{BusinessResponse: NewBusinessResponse(&item.Business), Role: item.Role})
	}
	httpadapter.WriteJSON(w, http.StatusOK, response)
}

// Applies the same owner/admin policy to business mutation and subscription reads.
func (h *BusinessHandlerImpl) RequireManagement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
		if !ok {
			return
		}
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		if err := h.s.RequireManager(r.Context(), userID, businessID); err != nil {
			httpadapter.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}
