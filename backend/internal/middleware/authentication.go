package middleware

import (
	"net/http"

	userapp "QueueLite/internal/user/app"

	"github.com/google/uuid"
)

func AuthMiddleware(userRepo userapp.UserRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				if err == http.ErrNoCookie {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}

			token := cookie.Value

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
