package inbound

import (
	"QueueLite/internal/business/app"
	"QueueLite/internal/business/domain"
	"QueueLite/internal/middleware"
	userapp "QueueLite/internal/user/app"
	userdomain "QueueLite/internal/user/domain"
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

type accountBusinessRepo struct {
	app.BusinessRepo
	userapp.UserRepo
	user             userdomain.User
	items            []domain.BusinessMembership
	role             string
	failure          error
	queried          uuid.UUID
	updates          int
	input            domain.UpdateBusiness
	owner            uuid.UUID
	created          domain.Business
	member           domain.MembershipUser
	memberRole       string
	memberIdentifier string
	memberByEmail    bool
	upsertedRole     string
}

func (r *accountBusinessRepo) CreateBusiness(_ context.Context, business *domain.Business) (*domain.Business, error) {
	business.ID = uuid.New()
	r.created = *business
	return business, r.failure
}
func (r *accountBusinessRepo) CreateUserBusinessRelation(_ context.Context, business, user uuid.UUID, role string) error {
	r.owner = user
	return nil
}

func (r *accountBusinessRepo) GetUser(_ context.Context, id uuid.UUID) (*userdomain.User, error) {
	if id != r.user.ID {
		return nil, userapp.ErrUserNotFound
	}
	return &r.user, nil
}
func (r *accountBusinessRepo) GetUserBusinesses(_ context.Context, id uuid.UUID) ([]domain.BusinessMembership, error) {
	r.queried = id
	return r.items, r.failure
}
func (r *accountBusinessRepo) GetUserBusinessRole(_ context.Context, user, business uuid.UUID) (string, error) {
	if user == r.user.ID {
		return r.role, r.failure
	}
	if user == r.member.ID {
		return r.memberRole, r.failure
	}
	return "", nil
}
func (r *accountBusinessRepo) FindMembershipUser(_ context.Context, identifier string, byEmail bool) (*domain.MembershipUser, error) {
	r.memberIdentifier, r.memberByEmail = identifier, byEmail
	if r.member.ID == uuid.Nil {
		return nil, domain.ErrMembershipUserNotFound
	}
	member := r.member
	return &member, r.failure
}
func (r *accountBusinessRepo) UpsertUserBusinessRelation(_ context.Context, _, user uuid.UUID, role string) error {
	if user != r.member.ID {
		return errors.New("wrong member")
	}
	r.upsertedRole = role
	return r.failure
}
func (r *accountBusinessRepo) UpdateBusiness(_ context.Context, _ uuid.UUID, input *domain.UpdateBusiness) error {
	r.updates++
	r.input = *input
	return r.failure
}

func accountBusinessRouter(t *testing.T, repo *accountBusinessRepo) (*chi.Mux, string) {
	t.Helper()
	t.Setenv("DATABASE_URL", "test")
	t.Setenv("SECRET", "test-secret")
	t.Setenv("REDIS_HOST", "test")
	handler := NewBusinessHandler(app.NewBusinessService(repo, nil))
	router := chi.NewRouter()
	router.Use(middleware.AuthMiddleware(repo))
	router.Get("/businesses/mine", handler.GetMyBusinesses)
	router.Post("/businesses/", handler.CreateBusiness)
	router.With(handler.RequireManagement).Put("/businesses/{businessID}", handler.UpdateBusiness)
	router.With(handler.RequireManagement).Put("/businesses/{businessID}/members", handler.UpsertBusinessMember)
	router.With(handler.RequireManagement).Delete("/businesses/{businessID}", handler.DeleteBusiness)
	router.With(handler.RequireManagement).Get("/subscriptions/businesses/{businessID}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	token, err := middleware.CreateToken(repo.user.Username, repo.user.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	return router, token
}

func TestCreateBusinessOwnerComesOnlyFromAuthenticatedContext(t *testing.T) {
	repo := &accountBusinessRepo{user: userdomain.User{ID: uuid.New(), Username: "Owner"}}
	router, token := accountBusinessRouter(t, repo)
	body := `{"name":"Clinic","location":"Jakarta","description":"Appointments","openTime":"22:00","closeTime":"06:00","email":"hello@example.test","phoneNumber":"081234567890"}`
	req := httptest.NewRequest("POST", "/businesses/?userId="+uuid.NewString(), strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != 201 || repo.owner != repo.user.ID || !repo.created.Operational {
		t.Fatalf("wrong owner/defaults: %d %s", res.Code, res.Body.String())
	}
	for _, test := range []struct {
		body, token string
		status      int
	}{{body, "", 401}, {`{"name":"Clinic","userId":"` + uuid.NewString() + `"}`, token, 400}, {strings.Replace(body, `"22:00"`, `"25:00"`, 1), token, 400}, {strings.Replace(body, `"name":"Clinic"`, `"name":" "`, 1), token, 400}} {
		req := httptest.NewRequest("POST", "/businesses/", strings.NewReader(test.body))
		if test.token != "" {
			req.AddCookie(&http.Cookie{Name: "token", Value: test.token})
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != test.status {
			t.Fatalf("status %d want %d: %s", res.Code, test.status, res.Body.String())
		}
	}
}
func TestMineUsesContextIdentityAndTypedBusinessResponse(t *testing.T) {
	repo := &accountBusinessRepo{user: userdomain.User{ID: uuid.New(), Username: "Owner"}}
	router, token := accountBusinessRouter(t, repo)
	for _, empty := range []bool{false, true} {
		repo.items = nil
		if !empty {
			repo.items = []domain.BusinessMembership{{Business: domain.Business{ID: uuid.New(), Name: "Owned", Operational: true, Email: "hello@example.test"}, Role: "owner"}}
		}
		req := httptest.NewRequest("GET", "/businesses/mine?userId="+uuid.NewString(), nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: token})
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != 200 || repo.queried != repo.user.ID {
			t.Fatalf("identity not scoped: %d %s", res.Code, res.Body.String())
		}
		var items []BusinessMembershipResponse
		if err := json.Unmarshal(res.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		if items == nil || len(items) != len(repo.items) {
			t.Fatalf("array contract: %s", res.Body.String())
		}
		if !empty && (items[0].Role != "owner" || items[0].Name != "Owned" || items[0].Email != "hello@example.test") {
			t.Fatalf("incomplete DTO: %+v", items)
		}
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest("GET", "/businesses/mine", nil))
	if res.Code != 401 {
		t.Fatal("guest accepted")
	}
	repo.failure = errors.New("private database failure")
	req := httptest.NewRequest("GET", "/businesses/mine", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != 500 || strings.Contains(res.Body.String(), "private database") {
		t.Fatalf("failure not masked: %s", res.Body.String())
	}
}
func TestBusinessManagementAuthorizationAndOperationalFalse(t *testing.T) {
	for _, role := range []string{"owner", "admin", "counter", "", "unexpected"} {
		t.Run(role, func(t *testing.T) {
			repo := &accountBusinessRepo{user: userdomain.User{ID: uuid.New(), Username: "Account"}, role: role}
			router, token := accountBusinessRouter(t, repo)
			request := httptest.NewRequest("PUT", "/businesses/"+uuid.NewString(), strings.NewReader(`{"name":"Updated","operational":false}`))
			request.AddCookie(&http.Cookie{Name: "token", Value: token})
			res := httptest.NewRecorder()
			router.ServeHTTP(res, request)
			if role == "owner" || role == "admin" {
				if res.Code != 200 || repo.updates != 1 || repo.input.Operational == nil || *repo.input.Operational || *repo.input.Name != "Updated" {
					t.Fatalf("manager update: %d %+v", res.Code, repo.input)
				}
			} else {
				if res.Code != 403 || repo.updates != 0 {
					t.Fatalf("unauthorized mutation: %d", res.Code)
				}
				for _, endpoint := range []struct{ method, path string }{{"DELETE", "/businesses/"}, {"GET", "/subscriptions/businesses/"}} {
					req := httptest.NewRequest(endpoint.method, endpoint.path+uuid.NewString(), nil)
					req.AddCookie(&http.Cookie{Name: "token", Value: token})
					res := httptest.NewRecorder()
					router.ServeHTTP(res, req)
					if res.Code != 403 {
						t.Fatalf("unguarded %s: %d", endpoint.path, res.Code)
					}
				}
			}
		})
	}
}

func TestBusinessMemberUpsertUsesManagerIdentityAndMinimalResponse(t *testing.T) {
	businessID := uuid.New()
	for _, scenario := range []struct {
		name, actorRole, requestedRole string
		status                         int
	}{
		{name: "owner-adds-admin", actorRole: "owner", requestedRole: "admin", status: http.StatusOK},
		{name: "admin-adds-counter", actorRole: "admin", requestedRole: "counter", status: http.StatusOK},
		{name: "admin-cannot-add-admin", actorRole: "admin", requestedRole: "admin", status: http.StatusForbidden},
		{name: "counter-denied", actorRole: "counter", requestedRole: "counter", status: http.StatusForbidden},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repo := &accountBusinessRepo{user: userdomain.User{ID: uuid.New(), Username: "Manager"}, role: scenario.actorRole, member: domain.MembershipUser{ID: uuid.New(), Username: "NewMember"}}
			router, token := accountBusinessRouter(t, repo)
			req := httptest.NewRequest(http.MethodPut, "/businesses/"+businessID.String()+"/members", strings.NewReader(`{"identifier":"person@example.test","role":"`+scenario.requestedRole+`"}`))
			req.AddCookie(&http.Cookie{Name: "token", Value: token})
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != scenario.status {
				t.Fatalf("status %d want %d: %s", res.Code, scenario.status, res.Body.String())
			}
			if scenario.status == http.StatusOK {
				if repo.memberIdentifier != "person@example.test" || !repo.memberByEmail || repo.upsertedRole != scenario.requestedRole {
					t.Fatalf("wrong lookup/upsert: %+v", repo)
				}
				var member map[string]any
				if err := json.Unmarshal(res.Body.Bytes(), &member); err != nil {
					t.Fatal(err)
				}
				if member["username"] != "NewMember" || member["role"] != scenario.requestedRole || len(member) != 3 {
					t.Fatalf("unsafe or incomplete response: %s", res.Body.String())
				}
			} else if repo.upsertedRole != "" {
				t.Fatal("unauthorized member mutation")
			}
		})
	}
}

func TestBusinessMemberProtectsSelfOwnerAndPeerAdminRoles(t *testing.T) {
	for _, scenario := range []struct {
		name, actorRole, memberRole string
		self                        bool
	}{
		{name: "self", actorRole: "owner", self: true},
		{name: "owner-relation", actorRole: "owner", memberRole: "owner"},
		{name: "admin-peer", actorRole: "admin", memberRole: "admin"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			actor := userdomain.User{ID: uuid.New(), Username: "Manager"}
			memberID := uuid.New()
			if scenario.self {
				memberID = actor.ID
			}
			repo := &accountBusinessRepo{user: actor, role: scenario.actorRole, member: domain.MembershipUser{ID: memberID, Username: "Member"}, memberRole: scenario.memberRole}
			router, token := accountBusinessRouter(t, repo)
			req := httptest.NewRequest(http.MethodPut, "/businesses/"+uuid.NewString()+"/members", strings.NewReader(`{"identifier":"Member","role":"counter"}`))
			req.AddCookie(&http.Cookie{Name: "token", Value: token})
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != http.StatusForbidden || repo.upsertedRole != "" {
				t.Fatalf("protected role changed: %d %s", res.Code, res.Body.String())
			}
		})
	}
}

func TestBusinessMemberUnknownAccountAndInvalidPayload(t *testing.T) {
	repo := &accountBusinessRepo{user: userdomain.User{ID: uuid.New(), Username: "Owner"}, role: "owner"}
	router, token := accountBusinessRouter(t, repo)
	for _, scenario := range []struct {
		body string
		want int
	}{
		{body: `{"identifier":"missing","role":"counter"}`, want: http.StatusNotFound},
		{body: `{"identifier":"","role":"counter"}`, want: http.StatusBadRequest},
		{body: `{"identifier":"person","role":"owner"}`, want: http.StatusBadRequest},
		{body: `{"identifier":"person","role":"counter","unexpected":true}`, want: http.StatusBadRequest},
	} {
		req := httptest.NewRequest(http.MethodPut, "/businesses/"+uuid.NewString()+"/members", strings.NewReader(scenario.body))
		req.AddCookie(&http.Cookie{Name: "token", Value: token})
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != scenario.want {
			t.Fatalf("status %d want %d: %s", res.Code, scenario.want, res.Body.String())
		}
	}
}

func TestManagementFailsClosedOnLookupFailure(t *testing.T) {
	repo := &accountBusinessRepo{user: userdomain.User{ID: uuid.New(), Username: "Account"}, role: "owner", failure: errors.New("private query error")}
	router, token := accountBusinessRouter(t, repo)
	req := httptest.NewRequest("PUT", "/businesses/"+uuid.NewString(), strings.NewReader(`{"operational":false}`))
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != 500 || repo.updates != 0 || strings.Contains(res.Body.String(), "private query") {
		t.Fatalf("lookup failed open: %d %s", res.Code, res.Body.String())
	}
}
