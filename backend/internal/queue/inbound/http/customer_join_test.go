package inbound_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/middleware"
	queuehttp "QueueLite/internal/queue/inbound/http"
	"QueueLite/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func queueRouter(t *testing.T, f *testutil.QueueFixture) http.Handler {
	t.Helper()
	t.Setenv("DATABASE_URL", "test")
	t.Setenv("SECRET", "queue-test-secret")
	t.Setenv("REDIS_HOST", "test")
	h := queuehttp.NewQueueHandler(f.Service)
	r := chi.NewRouter()
	r.Use(middleware.PublicMiddleware(f))
	r.Post("/queues/business/{businessID}/join", h.JoinBusinessQueue)
	r.Post("/queues/qr/{businessID}", h.RegisterQueueByQr)
	r.Get("/queues/qr/{businessID}/resolve", h.ResolveQueueQR)
	r.Post("/queues/", h.RegisterQueue)
	r.Group(func(r chi.Router) {
		r.Use(middleware.ProtectedMiddleware(*f.Service))
		r.Get("/queues/{queueID}", h.GetQueue)
		r.Get("/queues/{queueID}/state", h.GetQueueState)
		r.Delete("/queues/{queueID}", h.DeleteQueue)
	})
	return r
}
func request(r http.Handler, method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func ticketID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	if w.Code != 201 {
		t.Fatalf("join: %d %s", w.Code, w.Body)
	}
	var response queuehttp.QueueResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response.ID
}

func TestAuthenticatedJoinIgnoresSpoofedIdentityAndOwnsMultipleTickets(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	router := queueRouter(t, f)
	value, err := middleware.CreateToken(f.User.Username, f.User.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	auth := &http.Cookie{Name: "token", Value: value}
	response := request(router, "POST", "/queues/business/"+f.Business.ID.String()+"/join", `{"userId":"`+uuid.New().String()+`","priority":true,"username":"spoofed","phoneNumber":"123"}`, auth)
	id := ticketID(t, response)
	q, err := f.Service.GetQueue(context.Background(), uuid.MustParse(id))
	if err != nil || *q.UserID != f.User.ID || q.Priority {
		t.Fatal("browser controlled identity or priority")
	}
	// First ticket is persisted before joining another business.
	f.Persist(*q)
	f.Business.ID = uuid.New()
	second := request(router, "POST", "/queues/qr/"+f.Business.ID.String(), "", auth)
	id2 := ticketID(t, second)
	for _, ticket := range []string{id, id2} {
		for _, suffix := range []string{"", "/state"} {
			w := request(router, "GET", "/queues/"+ticket+suffix, "", auth)
			if w.Code != 200 {
				t.Fatalf("owner access without queueToken: %d %s", w.Code, w.Body)
			}
		}
	}
	other, err := middleware.CreateToken("Other", uuid.New().String())
	if err != nil {
		t.Fatal(err)
	}
	// PublicMiddleware checks existence; a valid different existing account is needed.
	f.User.ID = uuid.New()
	other, err = middleware.CreateToken("Other", f.User.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	w := request(router, "GET", "/queues/"+id, "", &http.Cookie{Name: "token", Value: other})
	if w.Code != 403 || !strings.Contains(w.Body.String(), "QUEUE_ACCESS_DENIED") {
		t.Fatalf("other owner: %d %s", w.Code, w.Body)
	}
}

func TestGuestCookiesOwnershipAndQRCompatibility(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	router := queueRouter(t, f)
	resolve := request(router, "GET", "/queues/qr/"+f.Business.ID.String()+"/resolve", "")
	if resolve.Code != 200 || !strings.Contains(resolve.Body.String(), `"requiresGuestForm":true`) {
		t.Fatal("QR resolve changed")
	}
	first := request(router, "POST", "/queues/qr/"+f.Business.ID.String(), `{"username":"Guest","phoneNumber":"081234567890"}`)
	id := ticketID(t, first)
	var firstCookies []*http.Cookie
	for _, c := range first.Result().Cookies() {
		if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != 86400 {
			t.Fatalf("cookie attributes: %#v", c)
		}
		if strings.HasSuffix(c.Name, id) {
			firstCookies = append(firstCookies, c)
		}
		if strings.HasPrefix(c.Name, "queueToken") {
			signed, err := middleware.ValidateQueueToken(c.Value)
			if err != nil || signed != id {
				t.Fatal("cookie UUID mismatch")
			}
		}
	}
	if len(firstCookies) != 2 {
		t.Fatal("missing per-ticket credentials")
	}
	f.Business.ID = uuid.New()
	second := request(router, "POST", "/queues/business/"+f.Business.ID.String()+"/join", `{"username":"Guest","phoneNumber":"081234567890"}`)
	id2 := ticketID(t, second)
	cookies := append(firstCookies, second.Result().Cookies()...)
	for _, ticket := range []string{id, id2} {
		for _, suffix := range []string{"", "/state"} {
			w := request(router, "GET", "/queues/"+ticket+suffix, "", cookies...)
			if w.Code != 200 {
				t.Fatalf("guest access: %d %s", w.Code, w.Body)
			}
		}
	}
	for _, bad := range [][]*http.Cookie{nil, {firstCookies[0]}, second.Result().Cookies(), {{Name: "queueToken", Value: "tampered"}, {Name: "guestToken", Value: "tampered"}}} {
		w := request(router, "GET", "/queues/"+id, "", bad...)
		if w.Code != 401 {
			t.Fatalf("invalid guest authorized: %d", w.Code)
		}
	}
	q, _ := f.Service.GetQueue(context.Background(), uuid.MustParse(id))
	f.Persist(*q)
	deleted := request(router, "DELETE", "/queues/"+id, "", firstCookies...)
	if deleted.Code != 200 {
		t.Fatalf("guest delete: %d %s", deleted.Code, deleted.Body)
	}
	w := request(router, "GET", "/queues/"+id, "", firstCookies...)
	if w.Code != 404 {
		t.Fatalf("deleted queue still accessible: %d", w.Code)
	}
}

func TestPublicGuestCannotSupplyUserID(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	router := queueRouter(t, f)
	for _, path := range []string{"/queues/business/" + f.Business.ID.String() + "/join", "/queues/qr/" + f.Business.ID.String(), "/queues/"} {
		body := `{"userId":"` + f.User.ID.String() + `","username":"Guest","phoneNumber":"081234567890"}`
		if path == "/queues/" {
			body = `{"businessId":"` + f.Business.ID.String() + `",` + body[1:]
		}
		w := request(router, "POST", path, body)
		if w.Code != 400 {
			t.Fatalf("guest userId accepted on %s: %d", path, w.Code)
		}
	}
	legacy := request(router, "POST", "/queues/", `{"businessId":"`+f.Business.ID.String()+`","username":"Guest","phoneNumber":"081234567890"}`)
	ticketID(t, legacy)
}

func TestMissingSigningConfigurationFailsBeforeJoin(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	router := queueRouter(t, f)
	t.Setenv("DATABASE_URL", "")
	w := request(router, "POST", "/queues/business/"+f.Business.ID.String()+"/join", `{"username":"Guest","phoneNumber":"081234567890"}`)
	if w.Code != 500 || f.Capacity != 100 {
		t.Fatalf("signing failure after registration: %d %d", w.Code, f.Capacity)
	}
}

func TestLegacyGuestMappingAndOwnershipCacheFailure(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	router := queueRouter(t, f)
	ctx := context.Background()
	w := request(router, "POST", "/queues/qr/"+f.Business.ID.String(), `{"username":"Guest","phoneNumber":"081234567890"}`)
	id := ticketID(t, w)
	cookies := w.Result().Cookies()
	q, err := f.Service.GetQueue(ctx, uuid.MustParse(id))
	if err != nil {
		t.Fatal(err)
	}
	f.Persist(*q)
	if err := f.Redis.Set(ctx, cache.CustomerQueueKey(id), "broken private cache payload", 0).Err(); err != nil {
		t.Fatal(err)
	}
	w = request(router, "GET", "/queues/"+id, "", cookies...)
	if w.Code != 500 || !strings.Contains(w.Body.String(), "VERIFY_QUEUE_OWNERSHIP_ERROR") || strings.Contains(w.Body.String(), "private cache") {
		t.Fatalf("cache failure: %d %s", w.Code, w.Body)
	}
	f.Redis.Del(ctx, cache.CustomerQueueKey(id))
	legacy := cache.GuestQueueUser{GuestID: q.UserID.String(), BusinessID: q.BusinessID.String(), QueueID: id, Username: "Guest", PhoneNumber: "6281234567890"}
	if err := cache.SetGuestQueueUser(ctx, f.Redis, legacy); err != nil {
		t.Fatal(err)
	}
	f.Redis.Set(ctx, cache.GuestPhoneKey(legacy.BusinessID, legacy.PhoneNumber), legacy.GuestID, 0)
	w = request(router, "GET", "/queues/"+id, "", cookies...)
	if w.Code != 200 {
		t.Fatalf("legacy ownership: %d %s", w.Code, w.Body)
	}
	if err := f.Service.MarkAsDone(ctx, q.ID); err != nil {
		t.Fatal(err)
	}
	if f.Server.Exists(cache.GuestPhoneKey(legacy.BusinessID, legacy.PhoneNumber)) {
		t.Fatal("legacy terminal phone retained")
	}
}
