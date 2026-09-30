package middleware

import (
	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
)

// RequireSelfUser is used after AuthMiddleware for account-specific requests.
func RequireSelfUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value, ok := UserIdFromContext(r.Context())
		current, err := uuid.Parse(value)
		if !ok || err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		target, err := uuid.Parse(chi.URLParam(r, "userID"))
		if err != nil {
			httpadapter.WriteError(w, apperror.New(apperror.KindInvalid, "INVALID_USER_ID", "invalid user id"))
			return
		}
		if current != target {
			httpadapter.WriteError(w, apperror.New(apperror.KindForbidden, "USER_ACCESS_DENIED", "you cannot manage another account"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
