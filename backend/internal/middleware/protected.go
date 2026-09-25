package middleware

import (
	"QueueLite/internal/queue/app"
	"net/http"

	"github.com/google/uuid"
)

func ProtectedMiddleware(queueService app.QueueService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("queueToken")
			if err != nil {
				if err == http.ErrNoCookie {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}

			token := cookie.Value

			queueIDString, err := ValidateQueueToken(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			queueID, err := uuid.Parse(queueIDString)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if _, err := queueService.GetQueue(r.Context(), queueID); err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			// inject the queue id to the context
			ctx := WithQueueId(r.Context(), queueIDString)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
