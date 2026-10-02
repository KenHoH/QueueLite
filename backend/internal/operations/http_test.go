package operations

import (
	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/counter/app"
	counterdomain "QueueLite/internal/counter/domain"
	counterhttp "QueueLite/internal/counter/inbound/http"
	"QueueLite/internal/middleware"
	queuedomain "QueueLite/internal/queue/domain"
	"QueueLite/internal/testutil"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type businessFixture struct {
	roles   map[uuid.UUID]map[uuid.UUID]string
	members map[uuid.UUID][]Member
	fail    bool
}

func (b *businessFixture) GetUserBusinessRole(_ context.Context, u, id uuid.UUID) (string, error) {
	if b.fail {
		return "", errors.New("database unavailable")
	}
	return b.roles[id][u], nil
}
func (b *businessFixture) ListBusinessMembers(_ context.Context, id uuid.UUID) ([]Member, error) {
	return b.members[id], nil
}

type counterFixture struct {
	app.CounterRepo
	items  map[uuid.UUID]*counterdomain.Counter
	listed uuid.UUID
}

func (c *counterFixture) GetCounter(_ context.Context, id uuid.UUID) (*counterdomain.Counter, error) {
	item := c.items[id]
	if item == nil {
		return nil, app.ErrCounterNotFound
	}
	copy := *item
	return &copy, nil
}
func (c *counterFixture) ListBusinessCounters(_ context.Context, id uuid.UUID) ([]counterdomain.Counter, error) {
	c.listed = id
	items := []counterdomain.Counter{}
	for _, c := range c.items {
		if c.BusinessID == id {
			items = append(items, *c)
		}
	}
	return items, nil
}
func (c *counterFixture) CreateCounter(_ context.Context, item *counterdomain.Counter) (*counterdomain.Counter, error) {
	item.ID = uuid.New()
	c.items[item.ID] = item
	return item, nil
}
func (c *counterFixture) UpdateCounter(_ context.Context, item *counterdomain.Counter) error {
	c.items[item.ID].Name = item.Name
	return nil
}
func (c *counterFixture) UpdateCounterEmployee(_ context.Context, id uuid.UUID, u *uuid.UUID) error {
	c.items[id].CurrentEmployeeID = u
	return nil
}
func (c *counterFixture) UpdateCounterCustomer(_ context.Context, id uuid.UUID, q *uuid.UUID) error {
	c.items[id].CurrentQueueID = q
	return nil
}
func (c *counterFixture) DeleteCounter(_ context.Context, id uuid.UUID) error {
	delete(c.items, id)
	return nil
}

type scenario struct {
	router                                                              *chi.Mux
	b                                                                   *businessFixture
	c                                                                   *counterFixture
	f                                                                   *testutil.QueueFixture
	business, other, owner, admin, staff, outsider, counter, unassigned uuid.UUID
}

func setup(t *testing.T) *scenario {
	f := testutil.NewQueueFixture(t)
	s := &scenario{f: f, business: f.Business.ID, other: uuid.New(), owner: uuid.New(), admin: uuid.New(), staff: uuid.New(), outsider: uuid.New(), counter: uuid.New(), unassigned: uuid.New()}
	s.b = &businessFixture{roles: map[uuid.UUID]map[uuid.UUID]string{s.business: {s.owner: "owner", s.admin: "admin", s.staff: "counter"}, s.other: {s.outsider: "owner"}}, members: map[uuid.UUID][]Member{s.business: {{s.owner, "Owner", "owner"}, {s.admin, "Admin", "admin"}, {s.staff, "Staff", "counter"}}}}
	s.c = &counterFixture{items: map[uuid.UUID]*counterdomain.Counter{s.counter: {ID: s.counter, BusinessID: s.business, Name: "Desk A", CurrentEmployeeID: &s.staff}, s.unassigned: {ID: s.unassigned, BusinessID: s.business, Name: "Desk B"}, uuid.New(): {ID: uuid.New(), BusinessID: s.other, Name: "Other"}}}
	h := New(s.b, s.c, f, f.Redis)
	counters := counterhttp.NewCounterHandler(app.NewCounterService(s.c, f, nil, f.Redis))
	r := chi.NewRouter()
	r.Get("/businesses/{businessID}/counters", h.ListCounters)
	r.Get("/businesses/{businessID}/members", h.ListMembers)
	r.With(h.RequireBusiness).Get("/businesses/{businessID}/queues", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	r.With(h.RequireCreate).Post("/counters/", counters.CreateCounter)
	r.Group(func(r chi.Router) {
		r.Use(h.RequireCounter)
		r.Get("/counters/{counterID}", counters.GetCounter)
		r.Put("/counters/{counterID}", counters.UpdateCounter)
		r.Delete("/counters/{counterID}", counters.DeleteCounter)
		r.Post("/counters/{counterID}/business/{businessID}/call-next", counters.CallNextQueue)
		r.Post("/counters/{counterID}/queues/{queueID}/process", counters.ProcessCalledQueue)
		r.Post("/counters/{counterID}/queues/{queueID}/skip", counters.SkipQueue)
		r.Delete("/counters/{counterID}/queues/{queueID}", counters.RemoveQueueFromCounter)
	})
	s.router = r
	return s
}
func (s *scenario) request(user uuid.UUID, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != uuid.Nil {
		r = r.WithContext(middleware.WithUserId(r.Context(), user.String()))
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}
func status(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("status %d want %d: %s", w.Code, want, w.Body.String())
	}
}

func TestBusinessListingsAndMembershipScope(t *testing.T) {
	s := setup(t)
	path := "/businesses/" + s.business.String()
	for _, u := range []uuid.UUID{s.owner, s.admin, s.staff} {
		w := s.request(u, "GET", path+"/counters", "")
		status(t, w, 200)
		var counters []counterhttp.CounterResponse
		if err := json.Unmarshal(w.Body.Bytes(), &counters); err != nil {
			t.Fatal(err)
		}
		want := 2
		if u == s.staff {
			want = 1
		}
		if len(counters) != want || s.c.listed != s.business {
			t.Fatalf("wrong scoped counters: %s", w.Body.String())
		}
		for _, c := range counters {
			if c.BusinessID != s.business.String() {
				t.Fatal("cross-business leak")
			}
		}
		status(t, s.request(u, "GET", path+"/queues", ""), 200)
		memberStatus := 200
		if u == s.staff {
			memberStatus = 403
		}
		status(t, s.request(u, "GET", path+"/members", ""), memberStatus)
	}
	for _, suffix := range []string{"/counters", "/members", "/queues"} {
		status(t, s.request(s.outsider, "GET", path+suffix, ""), 403)
		status(t, s.request(uuid.Nil, "GET", path+suffix, ""), 401)
	}
	w := s.request(s.owner, "GET", path+"/members", "")
	if strings.Contains(w.Body.String(), "email") || strings.Contains(w.Body.String(), "phone") || !strings.Contains(w.Body.String(), "Staff") {
		t.Fatal("member DTO contains wrong information")
	}
	s.b.fail = true
	status(t, s.request(s.owner, "GET", path+"/counters", ""), 500)
}
func TestCounterManagementRoleMatrix(t *testing.T) {
	for _, role := range []string{"owner", "admin", "counter", "unrelated"} {
		t.Run(role, func(t *testing.T) {
			s := setup(t)
			u := map[string]uuid.UUID{"owner": s.owner, "admin": s.admin, "counter": s.staff, "unrelated": s.outsider}[role]
			want := 200
			if role == "counter" || role == "unrelated" {
				want = 403
			}
			path := "/counters/" + s.unassigned.String()
			body := `{"businessId":"` + s.business.String() + `","name":"New desk"}`
			created := s.request(u, "POST", "/counters/", body)
			createWant := want
			if want == 200 {
				createWant = 201
			}
			status(t, created, createWant)
			status(t, s.request(u, "PUT", path, `{"name":"Renamed","currentEmployeeId":"`+s.staff.String()+`"}`), want)
			if want == 200 && (s.c.items[s.unassigned].Name != "Renamed" || *s.c.items[s.unassigned].CurrentEmployeeID != s.staff) {
				t.Fatal("update not applied")
			}
			status(t, s.request(u, "DELETE", path, ""), want)
		})
	}
}
func TestAssignmentValidationPrecedesAnyMutation(t *testing.T) {
	s := setup(t)
	path := "/counters/" + s.counter.String()
	status(t, s.request(s.owner, "PUT", path, `{"name":"Should not change","currentEmployeeId":"`+s.outsider.String()+`"}`), 403)
	if s.c.items[s.counter].Name != "Desk A" {
		t.Fatal("partial update before assignment validation")
	}
	status(t, s.request(s.owner, "PUT", path, `{"currentQueueId":"`+uuid.NewString()+`"}`), 400)
	status(t, s.request(s.owner, "PUT", path, `{"currentEmployeeId":""}`), 200)
	if s.c.items[s.counter].CurrentEmployeeID != nil {
		t.Fatal("assignment did not clear")
	}
	status(t, s.request(s.owner, "POST", "/counters/", `{"businessId":"`+s.business.String()+`","name":"Desk","currentEmployeeId":"`+s.outsider.String()+`"}`), 403)
}
func TestCounterResourceAuthorizationAndStaleState(t *testing.T) {
	s := setup(t)
	path := "/counters/" + s.counter.String()
	for _, u := range []uuid.UUID{s.owner, s.admin, s.staff} {
		status(t, s.request(u, "GET", path, ""), 200)
	}
	status(t, s.request(s.staff, "GET", "/counters/"+s.unassigned.String(), ""), 403)
	for _, suffix := range []string{"", "/business/" + s.business.String() + "/call-next", "/queues/" + uuid.NewString() + "/process", "/queues/" + uuid.NewString() + "/skip"} {
		method := "POST"
		if suffix == "" {
			method = "GET"
		}
		status(t, s.request(s.outsider, method, path+suffix, ""), 403)
	}
	status(t, s.request(s.owner, "POST", path+"/business/"+s.other.String()+"/call-next", ""), 403)
	queueID := uuid.New()
	s.c.items[s.counter].CurrentQueueID = &queueID
	status(t, s.request(s.owner, "POST", path+"/business/"+s.business.String()+"/call-next", ""), 409)
	status(t, s.request(s.owner, "DELETE", path, ""), 409)
	status(t, s.request(s.staff, "POST", path+"/queues/"+uuid.NewString()+"/process", ""), 409)
	status(t, s.request(s.staff, "POST", path+"/queues/"+queueID.String()+"/process", ""), 404)
	s.f.Persist(queuedomain.Queue{ID: queueID, BusinessID: s.other, CalledByCounterID: &s.counter, State: queuedomain.QueueStateCalled})
	status(t, s.request(s.owner, "POST", path+"/queues/"+queueID.String()+"/process", ""), 403)
}
func TestCounterMutationLockRejectsConcurrentAction(t *testing.T) {
	s := setup(t)
	s.f.Server.Set("counter:operation:"+s.counter.String(), "another-request")
	status(t, s.request(s.staff, "POST", "/counters/"+s.counter.String()+"/business/"+s.business.String()+"/call-next", ""), 409)
}

func TestAuthorizedCounterWorkflow(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	path := "/counters/" + s.counter.String()
	call := path + "/business/" + s.business.String() + "/call-next"
	w := s.request(s.staff, "POST", call, "")
	status(t, w, 200)
	if !strings.Contains(w.Body.String(), `"queue":null`) {
		t.Fatal("no waiting queue response")
	}
	q := queuedomain.Queue{ID: uuid.New(), BusinessID: s.business, Name: "Q001", State: queuedomain.QueueStateWaiting, Priority: true}
	s.f.Persist(q)
	if err := cache.AddWaitingQueue(ctx, s.f.Redis, q); err != nil {
		t.Fatal(err)
	}
	status(t, s.request(s.staff, "POST", call, ""), 200)
	stored, _ := s.f.GetQueue(ctx, q.ID)
	if stored.State != queuedomain.QueueStateCalled || !stored.Priority || s.c.items[s.counter].CurrentQueueID == nil {
		t.Fatal("call did not select priority queue")
	}
	// A duplicate call cannot silently complete the customer.
	status(t, s.request(s.staff, "POST", call, ""), 409)
	queuePath := path + "/queues/" + q.ID.String()
	status(t, s.request(s.staff, "DELETE", queuePath, ""), 409)
	status(t, s.request(s.staff, "POST", queuePath+"/process", ""), 200)
	status(t, s.request(s.staff, "POST", queuePath+"/process", ""), 409)
	status(t, s.request(s.staff, "DELETE", queuePath, ""), 200)
	stored, _ = s.f.GetQueue(ctx, q.ID)
	if stored.State != queuedomain.QueueStateCompleted || s.c.items[s.counter].CurrentQueueID != nil {
		t.Fatal("completion did not clear counter")
	}
	q.ID = uuid.New()
	q.Name = "Q002"
	s.f.Persist(q)
	if err := cache.AddWaitingQueue(ctx, s.f.Redis, q); err != nil {
		t.Fatal(err)
	}
	status(t, s.request(s.staff, "POST", call, ""), 200)
	status(t, s.request(s.staff, "POST", path+"/queues/"+q.ID.String()+"/skip", ""), 200)
	stored, _ = s.f.GetQueue(ctx, q.ID)
	if stored.State != queuedomain.QueueStateSkipped || s.c.items[s.counter].CurrentQueueID != nil {
		t.Fatal("skip did not clear counter")
	}
}
