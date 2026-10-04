package app

import (
	"QueueLite/internal/business/domain"
	counterapp "QueueLite/internal/counter/app"
	counterdomain "QueueLite/internal/counter/domain"
	subscriptionapp "QueueLite/internal/subscription/app"
	subscriptiondomain "QueueLite/internal/subscription/domain"
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"
)

type setupBusinessRepo struct {
	BusinessRepo
	business domain.Business
	owner    uuid.UUID
	role     string
}

func (r *setupBusinessRepo) CreateBusiness(_ context.Context, b *domain.Business) (*domain.Business, error) {
	b.ID = uuid.New()
	r.business = *b
	return b, nil
}
func (r *setupBusinessRepo) CreateUserBusinessRelation(_ context.Context, business, user uuid.UUID, role string) error {
	if business != r.business.ID {
		return errors.New("wrong business")
	}
	r.owner = user
	r.role = role
	return nil
}

type setupSubscriptionRepo struct {
	subscriptionapp.SubscriptionRepo
	subscription subscriptiondomain.Subscription
	plan         subscriptiondomain.BusinessPlan
}

func (r *setupSubscriptionRepo) CreateSubscription(_ context.Context, s *subscriptiondomain.Subscription, b *subscriptiondomain.BusinessPlan, u *subscriptiondomain.UserPlan) (*subscriptiondomain.Subscription, error) {
	r.subscription = *s
	r.plan = *b
	return s, nil
}

type setupCounterRepo struct {
	counterapp.CounterRepo
	counter counterdomain.Counter
	failure error
}

func (r *setupCounterRepo) CreateCounter(_ context.Context, c *counterdomain.Counter) (*counterdomain.Counter, error) {
	r.counter = *c
	return c, r.failure
}
func TestRegisterBusinessCreatesBackendDefaultsAndSurfacesCounterFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		repo := &setupBusinessRepo{}
		subscriptions := &setupSubscriptionRepo{}
		counters := &setupCounterRepo{}
		if fail {
			counters.failure = errors.New("counter failure")
		}
		service := NewBusinessService(repo, subscriptionapp.NewSubscriptionService(subscriptions), counterapp.NewCounterService(counters, nil, nil, nil))
		open, _ := time.Parse("15:04", "09:00")
		close, _ := time.Parse("15:04", "21:00")
		owner := uuid.New()
		business, err := service.RegisterBusiness(context.Background(), owner, domain.Business{Name: " Northside ", Location: " Jakarta ", Email: " hello@example.test ", PhoneNumber: "081234567890", Operational: true, OpenTime: open, CloseTime: close})
		if (err != nil) != fail {
			t.Fatalf("setup error: %v", err)
		}
		if !fail && (business.ID == uuid.Nil || business.Name != "Northside") {
			t.Fatal("business not created/trimmed")
		}
		if repo.owner != owner || repo.role != "owner" || subscriptions.subscription.BusinessID == nil || *subscriptions.subscription.BusinessID != repo.business.ID || subscriptions.plan.BusinessPlanType != subscriptiondomain.BusinessPlanTypeFree || counters.counter.BusinessID != repo.business.ID || counters.counter.Name != "Default Counter" {
			t.Fatal("backend setup defaults missing")
		}
	}
}
