package inbound

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/business/app"
	"QueueLite/internal/business/domain"

	"QueueLite/internal/httputil"

	"github.com/google/uuid"
)

type BusinessHandlerImpl struct {
	s *app.BusinessService
}

func NewBusinessHandler(s *app.BusinessService) *BusinessHandlerImpl {
	return &BusinessHandlerImpl{s: s}
}

func (h *BusinessHandlerImpl) CreateBusiness(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := httputil.CurrentUserID(w, r)
	if !ok {
		return
	}
	var request CreateBusinessRequest
	if err := httputil.DecodeJSON(r, &request); err != nil {
		httputil.WriteInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	openTime, ok := httputil.ParseBusinessTime(w, request.OpenTime, "INVALID_OPEN_TIME", "invalid open time")
	if !ok {
		return
	}
	closeTime, ok := httputil.ParseBusinessTime(w, request.CloseTime, "INVALID_CLOSE_TIME", "invalid close time")
	if !ok {
		return
	}

	business, err := h.s.RegisterBusiness(r.Context(), ownerID, domain.Business{
		Name:        request.Name,
		Location:    request.Location,
		Description: request.Description,
		Operational: true,
		OpenTime:    openTime,
		CloseTime:   closeTime,
		Email:       request.Email,
		PhoneNumber: request.PhoneNumber,
	})
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusCreated, NewBusinessResponse(business))
}

func (h *BusinessHandlerImpl) GetBusiness(w http.ResponseWriter, r *http.Request) {
	businessID, ok := httputil.ParseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}

	business, err := h.s.GetBusiness(r.Context(), businessID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, NewBusinessResponse(business))
}

func (h *BusinessHandlerImpl) GetBusinessAll(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	businesses, nextCursor, err := h.s.GetAllBusiness(r.Context(), parseBusinessCursor(r), limit)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]any{
		"data":       NewBusinessResponses(businesses),
		"nextCursor": newBusinessCursorResponse(nextCursor),
	})
}

func (h *BusinessHandlerImpl) SearchBusiness(w http.ResponseWriter, r *http.Request) {
	businesses, err := h.s.SearchBusiness(r.Context(), r.URL.Query().Get("name"))
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, NewBusinessResponses(businesses))
}

func (h *BusinessHandlerImpl) UpdateBusiness(w http.ResponseWriter, r *http.Request) {
	businessID, ok := httputil.ParseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}

	var request UpdateBusinessRequest
	if err := httputil.DecodeJSON(r, &request); err != nil {
		httputil.WriteInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	input := domain.UpdateBusiness{
		Operational: request.Operational,
		Name:        request.Name,
		Location:    request.Location,
		Description: request.Description,
		Email:       request.Email,
		PhoneNumber: request.PhoneNumber,
	}
	if request.OpenTime != nil {
		openTime, ok := httputil.ParseBusinessTime(w, *request.OpenTime, "INVALID_OPEN_TIME", "invalid open time")
		if !ok {
			return
		}
		input.OpenTime = &openTime
	}
	if request.CloseTime != nil {
		closeTime, ok := httputil.ParseBusinessTime(w, *request.CloseTime, "INVALID_CLOSE_TIME", "invalid close time")
		if !ok {
			return
		}
		input.CloseTime = &closeTime
	}
	if input.Operational == nil && input.Name == nil && input.Location == nil && input.Description == nil && input.OpenTime == nil && input.CloseTime == nil && input.Email == nil && input.PhoneNumber == nil {
		httputil.WriteInvalid(w, "NO_BUSINESS_FIELDS", "no business fields provided")
		return
	}

	if err := h.s.UpdateBusiness(r.Context(), businessID, input); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "business updated"})
}

func (h *BusinessHandlerImpl) DeleteBusiness(w http.ResponseWriter, r *http.Request) {
	businessID, ok := httputil.ParseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}

	if err := h.s.DeleteBusiness(r.Context(), businessID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "business deleted"})
}

func parseBusinessCursor(r *http.Request) *domain.BusinessCursor {
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
	return &domain.BusinessCursor{CreatedAt: createdAt, ID: id}
}

func newBusinessCursorResponse(cursor *domain.BusinessCursor) map[string]string {
	if cursor == nil {
		return nil
	}
	return map[string]string{
		"createdAt": cursor.CreatedAt.Format(time.RFC3339),
		"id":        cursor.ID.String(),
	}
}

func (h *BusinessHandlerImpl) UpsertBusinessMember(w http.ResponseWriter, r *http.Request) {
	businessID, ok := httputil.ParseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}
	actorID, ok := httputil.CurrentUserID(w, r)
	if !ok {
		return
	}
	var request UpsertBusinessMemberRequest
	if err := httputil.DecodeJSON(r, &request); err != nil {
		httputil.WriteInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	member, err := h.s.UpsertBusinessMember(r.Context(), actorID, businessID, request.Identifier, request.Role)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, member)
}

func (h *BusinessHandlerImpl) GetMyBusinesses(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.CurrentUserID(w, r)
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
		businessID, ok := httputil.ParseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
		if !ok {
			return
		}
		userID, ok := httputil.CurrentUserID(w, r)
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
