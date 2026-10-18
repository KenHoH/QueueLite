package inbound

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"QueueLite/internal/subscription/app"
	"QueueLite/internal/subscription/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type updateUserPlanMockRepo struct {
	app.SubscriptionRepo
	updatedPlan domain.UserPlan
	err         error
}

func (m *updateUserPlanMockRepo) UpdateUserSubscriptionPlanType(_ context.Context, _ uuid.UUID, userPlan domain.UserPlan) error {
	m.updatedPlan = userPlan
	return m.err
}

func TestUpdateUserSubscriptionTypeSuccess(t *testing.T) {
	userID := uuid.New()
	mockRepo := &updateUserPlanMockRepo{}
	service := app.NewSubscriptionService(mockRepo)
	handler := NewSubscriptionHandler(service)

	r := chi.NewRouter()
	r.Patch("/updateSubscription/{userID}", handler.UpdateUserSubscriptionType)

	body := `{"userPlanType": "premium"}`
	req := httptest.NewRequest(http.MethodPatch, "/updateSubscription/"+userID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "user subscription plan updated") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
	if mockRepo.updatedPlan.UserPlanType != domain.UserPlanTypePremium {
		t.Fatalf("expected plan type premium, got %s", mockRepo.updatedPlan.UserPlanType)
	}
	if mockRepo.updatedPlan.Slots != 4 {
		t.Fatalf("expected 4 slots for premium, got %d", mockRepo.updatedPlan.Slots)
	}
}

func TestUpdateUserSubscriptionTypeValidation(t *testing.T) {
	userID := uuid.New()
	mockRepo := &updateUserPlanMockRepo{}
	service := app.NewSubscriptionService(mockRepo)
	handler := NewSubscriptionHandler(service)

	r := chi.NewRouter()
	r.Patch("/updateSubscription/{userID}", handler.UpdateUserSubscriptionType)

	body := `{"userPlanType": "invalid_plan"}`
	req := httptest.NewRequest(http.MethodPatch, "/updateSubscription/"+userID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}
