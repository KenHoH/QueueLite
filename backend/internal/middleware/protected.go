package middleware

import (
	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	"QueueLite/internal/queue/app"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
)

func ProtectedMiddleware(queueService app.QueueService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			queueID, err := uuid.Parse(chi.URLParam(r, "queueID"))
			if err != nil {
				httpadapter.WriteError(w, apperror.New(apperror.KindInvalid, "INVALID_QUEUE_ID", "invalid queue id"))
				return
			}
			userID, authenticated := OptionalUserIdFromContext(r.Context())
			credential := hasQueueCredential(r, queueID.String())
			if !authenticated && !credential {
				httpadapter.WriteError(w, apperror.New(apperror.KindUnauthorized, "QUEUE_ACCESS_REQUIRED", "queue ownership credentials required"))
				return
			}
			queue, err := queueService.GetQueue(r.Context(), queueID)
			if err != nil {
				httpadapter.WriteError(w, err)
				return
			}
			owned := authenticated && queue.UserID != nil && queue.UserID.String() == userID
			if !owned && credential {
				// Scoped and legacy root cookies can have duplicate names. Check all values.
				for _, cookie := range r.Cookies() {
					if cookie.Name != "guestToken" && cookie.Name != "guestToken_"+queueID.String() {
						continue
					}
					guestID, businessID, phone, err := ValidateGuestToken(cookie.Value)
					if err != nil {
						continue
					}
					guestOwns, accessErr := queueService.GuestOwnsQueue(r.Context(), queue, guestID, businessID, phone)
					if accessErr != nil {
						httpadapter.WriteError(w, accessErr)
						return
					}
					if guestOwns {
						owned = true
						break
					}
				}
			}
			if !owned {
				kind, code := apperror.KindUnauthorized, "QUEUE_ACCESS_REQUIRED"
				if authenticated {
					kind, code = apperror.KindForbidden, "QUEUE_ACCESS_DENIED"
				}
				httpadapter.WriteError(w, apperror.New(kind, code, "queue ownership credentials required"))
				return
			}
			next.ServeHTTP(w, r.WithContext(WithQueueId(r.Context(), queueID.String())))
		})
	}
}

func hasQueueCredential(r *http.Request, queueID string) bool {
	for _, cookie := range r.Cookies() {
		if cookie.Name == "queueToken" || cookie.Name == "queueToken_"+queueID {
			id, err := ValidateQueueToken(cookie.Value)
			if err == nil && id == queueID {
				return true
			}
		}
	}
	return false
}
