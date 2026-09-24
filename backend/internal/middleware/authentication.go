package middleware

import (
	"net/http"
	"strings"

	userapp "QueueLite/internal/user/app"

	"github.com/google/uuid"
)

func AuthMiddleware(userRepo userapp.UserRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimSpace(r.Header.Get("Authorization"))
			if token == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			token = strings.TrimPrefix(token, "Bearer ")

			userIDString, err := ValidateToken(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			userID, err := uuid.Parse(userIDString)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if _, err := userRepo.GetUser(r.Context(), userID); err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			// inject the user id to the context
			ctx := WithUserId(r.Context(), userIDString)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
