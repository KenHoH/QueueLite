package middleware

import (
	"net/http"

	userapp "QueueLite/internal/user/app"

	"github.com/google/uuid"
)

func PublicMiddleware(userRepo userapp.UserRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			userIDString, err := ValidateToken(cookie.Value)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			userID, err := uuid.Parse(userIDString)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if _, err := userRepo.GetUser(r.Context(), userID); err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := WithOptionalUserId(r.Context(), userIDString)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
