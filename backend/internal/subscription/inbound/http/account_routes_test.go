package inbound

import (
	"QueueLite/internal/middleware"
	"QueueLite/internal/subscription/app"
	"QueueLite/internal/subscription/domain"
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type subscriptionAccessRepo struct {
	app.SubscriptionRepo
	item    domain.Subscription
	role    string
	failure bool
}

func (r *subscriptionAccessRepo) GetSubscription(_ context.Context, id uuid.UUID) (*domain.Subscription, error) {
	if id != r.item.ID {
		return nil, app.ErrSubscriptionNotFound
	}
	copy := r.item
	return &copy, nil
}
func (r *subscriptionAccessRepo) GetUserBusinessRole(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	if r.failure {
		return "", errors.New("private database failure")
	}
	return r.role, nil
}
func TestAccountSubscriptionRoutesDenyAdministration(t *testing.T) {
	id, user := uuid.New(), uuid.New()
	repo := &subscriptionAccessRepo{item: domain.Subscription{ID: id, UserID: &user}}
	router := chi.NewRouter()
	RegisterAccountRoutes(router, NewSubscriptionHandler(app.NewSubscriptionService(repo)), func(next http.Handler) http.Handler { return next }, repo)
	routes := []struct{ method, path string }{
		{"GET", "/"}, {"POST", "/users/" + user.String() + "/use"}, {"PATCH", "/users/" + user.String() + "/slots"},
		{"PATCH", "/businesses/" + uuid.NewString() + "/capacity/decrease"}, {"PUT", "/" + id.String()},
		{"PATCH", "/" + id.String() + "/time"}, {"PATCH", "/" + id.String() + "/activate"}, {"PATCH", "/" + id.String() + "/deactivate"}, {"DELETE", "/" + id.String()},
	}
	for _, route := range routes {
		t.Run(route.method+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, strings.NewReader(`{"amount":9999}`))
			req = req.WithContext(middleware.WithUserId(req.Context(), user.String()))
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != 403 || !strings.Contains(res.Body.String(), "SUBSCRIPTION_ADMIN_REQUIRED") {
				t.Fatalf("unsafe administration: %d %s", res.Code, res.Body)
			}
		})
	}
}
func TestSubscriptionIDReadChecksOwnershipAndBusinessRoles(t *testing.T) {
	user, business := uuid.New(), uuid.New()
	for _, scenario := range []struct {
		name, role            string
		business, other, fail bool
		status                int
	}{
		{name: "own-user", status: 200}, {name: "other-user", other: true, status: 403},
		{name: "owner", role: "owner", business: true, status: 200}, {name: "admin", role: "admin", business: true, status: 200},
		{name: "counter", role: "counter", business: true, status: 403}, {name: "unrelated", business: true, status: 403},
		{name: "lookup-failure", business: true, fail: true, status: 500},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repo := &subscriptionAccessRepo{item: domain.Subscription{ID: uuid.New(), UserID: &user, Type: domain.SubscriptionTypeUser, Status: domain.SubscriptionStatusActive, StartDate: time.Now()}, role: scenario.role, failure: scenario.fail}
			if scenario.business {
				repo.item.UserID = nil
				repo.item.BusinessID = &business
				repo.item.Type = domain.SubscriptionTypeBusiness
			}
			router := chi.NewRouter()
			RegisterAccountRoutes(router, NewSubscriptionHandler(app.NewSubscriptionService(repo)), func(next http.Handler) http.Handler { return next }, repo)
			who := user
			if scenario.other {
				who = uuid.New()
			}
			req := httptest.NewRequest("GET", "/"+repo.item.ID.String(), nil)
			req = req.WithContext(middleware.WithUserId(req.Context(), who.String()))
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != scenario.status || strings.Contains(res.Body.String(), "private database failure") {
				t.Fatalf("wrong authorization: %d %s", res.Code, res.Body)
			}
		})
	}
}
