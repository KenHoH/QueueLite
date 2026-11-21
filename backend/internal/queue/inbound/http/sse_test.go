package inbound_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/middleware"
	"QueueLite/internal/queue/domain"
	queuehttp "QueueLite/internal/queue/inbound/http"
	"QueueLite/internal/testutil"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type sseRecorder struct {
	header http.Header
	status int
	mu     sync.Mutex
	buf    strings.Builder
	notify chan struct{}
}

func newSSERecorder() *sseRecorder {
	return &sseRecorder{
		header: make(http.Header),
		status: http.StatusOK,
		notify: make(chan struct{}, 16),
	}
}

func (r *sseRecorder) Header() http.Header { return r.header }
func (r *sseRecorder) WriteHeader(code int) {
	r.mu.Lock()
	r.status = code
	r.mu.Unlock()
}
func (r *sseRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	n, err := r.buf.Write(p)
	r.mu.Unlock()
	select {
	case r.notify <- struct{}{}:
	default:
	}
	return n, err
}
func (r *sseRecorder) Flush() {}
func (r *sseRecorder) BodyString() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

func TestCustomerQueueSSEStreamsInitialAndLiveUpdatesAndClosesOnTerminal(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	t.Setenv("DATABASE_URL", "test")
	t.Setenv("SECRET", "sse-test-secret")
	t.Setenv("REDIS_HOST", "test")

	q := domain.Queue{
		ID:         uuid.New(),
		BusinessID: f.Business.ID,
		UserID:     &f.User.ID,
		Name:       "A001",
		State:      domain.QueueStateWaiting,
		CreatedAt:  time.Now().UTC(),
	}
	f.Persist(q)
	if err := cache.AddWaitingQueue(context.Background(), f.Redis, q); err != nil {
		t.Fatal(err)
	}

	h := queuehttp.NewQueueHandler(f.Service)
	r := chi.NewRouter()
	r.Use(middleware.PublicMiddleware(f))
	r.With(middleware.ProtectedMiddleware(*f.Service)).Get("/queues/{queueID}/events", h.CustomerQueueSSEHandler)

	token, err := middleware.CreateToken(f.User.Username, f.User.ID.String())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/queues/"+q.ID.String()+"/events", nil).WithContext(ctx)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})

	rec := newSSERecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.ServeHTTP(rec, req)
	}()

	// Wait for initial SSE status frame.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !strings.Contains(rec.BodyString(), `"state":"waiting"`) {
		time.Sleep(10 * time.Millisecond)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", ct)
	}
	if !strings.Contains(rec.BodyString(), "event: status") || !strings.Contains(rec.BodyString(), `"state":"waiting"`) {
		t.Fatalf("missing initial status frame: %s", rec.BodyString())
	}

	// Mark queue done; handler should push terminal status event and close the SSE stream automatically.
	if err := f.Service.MarkAsDone(context.Background(), q.ID); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("SSE stream did not close on terminal state; body=%s", rec.BodyString())
	}

	scanner := bufio.NewScanner(strings.NewReader(rec.BodyString()))
	statusEvents := 0
	for scanner.Scan() {
		if scanner.Text() == "event: status" {
			statusEvents++
		}
	}
	if statusEvents < 2 || !strings.Contains(rec.BodyString(), `"state":"done"`) {
		t.Fatalf("expected initial + terminal status events, got %d: %s", statusEvents, rec.BodyString())
	}
}

func TestBusinessQueueSSEStreamsConnectedAndUpdates(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	h := queuehttp.NewQueueHandler(f.Service)
	r := chi.NewRouter()
	r.Get("/businesses/{businessID}/events", h.BusinessQueueSSEHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/businesses/"+f.Business.ID.String()+"/events", nil).WithContext(ctx)

	rec := newSSERecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.ServeHTTP(rec, req)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !strings.Contains(rec.BodyString(), "event: connected") {
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(rec.BodyString(), "event: connected") {
		t.Fatalf("missing connected event: %s", rec.BodyString())
	}

	if err := cache.PublishMessage(context.Background(), f.Redis, cache.QueueEventChannel(f.Business.ID.String()), "queue.update"); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !strings.Contains(rec.BodyString(), "event: queue.update") {
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(rec.BodyString(), "event: queue.update") {
		t.Fatalf("missing queue.update event: %s", rec.BodyString())
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("business SSE did not stop on context cancel")
	}
}
