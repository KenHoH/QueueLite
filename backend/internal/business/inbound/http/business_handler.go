package inbound

import (
	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	"QueueLite/internal/business/app"
	"QueueLite/internal/business/domain"
	"QueueLite/internal/middleware"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type BusinessHandlerImpl struct {
	s *app.BusinessService
}

func NewBusinessHandler(s *app.BusinessService) *BusinessHandlerImpl {
	return &BusinessHandlerImpl{s: s}
}

func (h *BusinessHandlerImpl) CreateBusiness(w http.ResponseWriter, r *http.Request) {
	var request CreateBusinessRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	openTime, ok := parseBusinessTime(w, request.OpenTime, "INVALID_OPEN_TIME", "invalid open time")
	if !ok {
		return
	}
	closeTime, ok := parseBusinessTime(w, request.CloseTime, "INVALID_CLOSE_TIME", "invalid close time")
	if !ok {
		return
	}

	ownerID := uuid.Nil
	if ownerIDString, ok := middleware.UserIdFromContext(r.Context()); ok {
		parsed, err := uuid.Parse(ownerIDString)
		if err != nil {
			writeInvalid(w, "INVALID_OWNER_ID", "invalid owner id")
			return
		}
		ownerID = parsed
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
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
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
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}

	var request UpdateBusinessRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	input := domain.UpdateBusiness{
		Name:        request.Name,
		Location:    request.Location,
		Description: request.Description,
		Email:       request.Email,
		PhoneNumber: request.PhoneNumber,
	}
	if request.OpenTime != nil {
		openTime, ok := parseBusinessTime(w, *request.OpenTime, "INVALID_OPEN_TIME", "invalid open time")
		if !ok {
			return
		}
		input.OpenTime = &openTime
	}
	if request.CloseTime != nil {
		closeTime, ok := parseBusinessTime(w, *request.CloseTime, "INVALID_CLOSE_TIME", "invalid close time")
		if !ok {
			return
		}
		input.CloseTime = &closeTime
	}
	if input.Name == nil && input.Location == nil && input.Description == nil && input.OpenTime == nil && input.CloseTime == nil && input.Email == nil && input.PhoneNumber == nil {
		writeInvalid(w, "NO_BUSINESS_FIELDS", "no business fields provided")
		return
	}

	if err := h.s.UpdateBusiness(r.Context(), businessID, input); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "business updated"})
}

func (h *BusinessHandlerImpl) DeleteBusiness(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
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

func parseBusinessTime(w http.ResponseWriter, value string, code string, message string) (time.Time, bool) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		writeInvalid(w, code, message)
		return time.Time{}, false
	}
	return parsed, true
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, name string, code string, message string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, name)))
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
