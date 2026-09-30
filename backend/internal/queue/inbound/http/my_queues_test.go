package inbound_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"QueueLite/internal/middleware"
	"QueueLite/internal/queue/domain"
	queuehttp "QueueLite/internal/queue/inbound/http"
	"QueueLite/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestMyQueuesUsesAuthenticatedIdentityAndActiveStates(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	t.Setenv("DATABASE_URL", "test")
	t.Setenv("SECRET", "queue-test-secret")
	t.Setenv("REDIS_HOST", "test")
	other := uuid.New()
	for _, owner := range []uuid.UUID{f.User.ID, other} {
		for _, state := range []domain.QueueState{domain.QueueStateWaiting, domain.QueueStateCalled, domain.QueueStateProcessing, domain.QueueStateCompleted, domain.QueueStateSkipped, domain.QueueStateCancelled} {
			f.Persist(domain.Queue{ID: uuid.New(), UserID: &owner, BusinessID: f.Business.ID, Name: "A001", State: state})
		}
	}
	h := queuehttp.NewQueueHandler(f.Service)
	r := chi.NewRouter()
	r.With(middleware.AuthMiddleware(f)).Get("/queues/me", h.GetMyQueues)
	token, err := middleware.CreateToken(f.User.Username, f.User.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	w := request(r, "GET", "/queues/me?userId="+other.String(), `{"userId":"`+other.String()+`"}`, &http.Cookie{Name: "token", Value: token})
	if w.Code != 200 {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}
	var queues []queuehttp.QueueResponse
	if err := json.Unmarshal(w.Body.Bytes(), &queues); err != nil {
		t.Fatal(err)
	}
	if len(queues) != 3 {
		t.Fatalf("expected three active queues, got %v", queues)
	}
	states := map[string]bool{}
	for _, q := range queues {
		if q.UserID == nil || *q.UserID != f.User.ID.String() {
			t.Fatal("another owner's queue leaked")
		}
		states[q.State] = true
	}
	for _, state := range []string{"waiting", "called", "processing"} {
		if !states[state] {
			t.Fatalf("missing %s", state)
		}
	}
	for _, cookie := range []*http.Cookie{nil, {Name: "token", Value: "invalid"}} {
		req := httptest.NewRequest("GET", "/queues/me", nil)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		denied := httptest.NewRecorder()
		r.ServeHTTP(denied, req)
		if denied.Code != 401 {
			t.Fatalf("guest/tampered token: %d", denied.Code)
		}
	}
	f.Queues = map[uuid.UUID]domain.Queue{}
	empty := request(r, "GET", "/queues/me", "", &http.Cookie{Name: "token", Value: token})
	if empty.Code != 200 || strings.TrimSpace(empty.Body.String()) != "[]" {
		t.Fatalf("empty list: %d %s", empty.Code, empty.Body)
	}
}
