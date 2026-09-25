package middleware

import (
	"QueueLite/internal/queue/app"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func ProtectedMiddleware(queueService app.QueueService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimSpace(r.Header.Get("Authorization"))
			if token == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			token = strings.TrimPrefix(token, "Bearer ")

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
