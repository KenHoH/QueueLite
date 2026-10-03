package inbound

import (
	"context"
	"net/http"

	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	"QueueLite/internal/httputil"
	"QueueLite/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type BusinessRoleLookup interface {
	GetUserBusinessRole(context.Context, uuid.UUID, uuid.UUID) (string, error)
}

// Account HTTP routes expose owned reads. Plan/quota administration needs a
// trusted billing/admin identity; an ordinary login is not that authority.
func RegisterAccountRoutes(r chi.Router, h *SubscriptionHandlerImpl, management func(http.Handler) http.Handler, roles BusinessRoleLookup) {
	r.With(management).Get("/businesses/{businessID}", h.GetBusinessSubscriptionInfo)
	r.With(middleware.RequireSelfUser).Get("/users/{userID}", h.GetUserSubscriptionInfo)
	r.With(h.requireOwnedSubscription(roles)).Get("/{subscriptionID}", h.GetSubscription)
	r.Get("/", unavailableAdministration)
	r.Post("/users/{userID}/use", unavailableAdministration)
	r.Patch("/users/{userID}/slots", unavailableAdministration)
	r.Patch("/businesses/{businessID}/capacity/decrease", unavailableAdministration)
	r.Put("/{subscriptionID}", unavailableAdministration)
	r.Patch("/{subscriptionID}/time", unavailableAdministration)
	r.Patch("/{subscriptionID}/activate", unavailableAdministration)
	r.Patch("/{subscriptionID}/deactivate", unavailableAdministration)
	r.Delete("/{subscriptionID}", unavailableAdministration)
}

func unavailableAdministration(w http.ResponseWriter, r *http.Request) {
	httpadapter.WriteError(w, apperror.New(apperror.KindForbidden, "SUBSCRIPTION_ADMIN_REQUIRED", "subscription administration is unavailable for account sessions"))
}

func (h *SubscriptionHandlerImpl) requireOwnedSubscription(roles BusinessRoleLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := httputil.ParseUUIDParam(w, r, "subscriptionID", "INVALID_SUBSCRIPTION_ID", "invalid subscription id")
			if !ok {
				return
			}
			userString, authenticated := middleware.UserIdFromContext(r.Context())
			user, err := uuid.Parse(userString)
			if !authenticated || err != nil {
				httpadapter.WriteError(w, apperror.New(apperror.KindUnauthorized, "UNAUTHORIZED", "authentication required"))
				return
			}
			subscription, err := h.s.GetSubscription(r.Context(), id)
			if err != nil {
				httpadapter.WriteError(w, err)
				return
			}
			allowed := subscription.UserID != nil && *subscription.UserID == user
			if subscription.BusinessID != nil {
				role, err := roles.GetUserBusinessRole(r.Context(), user, *subscription.BusinessID)
				if err != nil {
					httpadapter.WriteError(w, apperror.Wrap(apperror.KindInternal, "BUSINESS_ACCESS_ERROR", "failed to check business access", err))
					return
				}
				allowed = role == "owner" || role == "admin"
			}
			if !allowed {
				httpadapter.WriteError(w, apperror.New(apperror.KindForbidden, "SUBSCRIPTION_ACCESS_DENIED", "subscription access denied"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
