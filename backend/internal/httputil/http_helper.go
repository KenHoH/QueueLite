package httputil

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	"QueueLite/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func CurrentUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	value, ok := middleware.UserIdFromContext(r.Context())
	id, err := uuid.Parse(value)
	if !ok || err != nil || id == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return uuid.Nil, false
	}
	return id, true
}

func ParseUUIDParam(w http.ResponseWriter, r *http.Request, name string, code string, message string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, name)))
	if err != nil {
		WriteInvalid(w, code, message)
		return uuid.Nil, false
	}
	return id, true
}

func WriteInvalid(w http.ResponseWriter, code string, message string) {
	httpadapter.WriteError(w, apperror.New(apperror.KindInvalid, code, message))
}

func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func ParseBusinessTime(w http.ResponseWriter, value string, code string, message string) (time.Time, bool) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		WriteInvalid(w, code, message)
		return time.Time{}, false
	}
	return parsed, true
}
