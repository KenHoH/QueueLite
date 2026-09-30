package inbound

import (
	"QueueLite/internal/middleware"
	"QueueLite/internal/user/app"
	"QueueLite/internal/user/domain"
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type profileRepo struct {
	app.UserRepo
	user    domain.User
	updates int
	failure error
}

func (p *profileRepo) GetUser(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if id != p.user.ID {
		return nil, app.ErrUserNotFound
	}
	user := p.user
	return &user, nil
}
func (p *profileRepo) UpdateUser(_ context.Context, user *domain.User) error {
	p.updates++
	if p.failure != nil {
		return p.failure
	}
	p.user = *user
	return nil
}
func TestProfileUpdateIsSelfOnlyAndPasswordsAreHashed(t *testing.T) {
	t.Setenv("DATABASE_URL", "test")
	t.Setenv("SECRET", "test-secret")
	t.Setenv("REDIS_HOST", "test")
	repo := &profileRepo{user: domain.User{ID: uuid.New(), Username: "Account", PhoneNumber: "081234567890"}}
	handler := NewUserHandler(app.NewUserService(repo, nil))
	router := chi.NewRouter()
	router.Use(middleware.AuthMiddleware(repo))
	router.With(middleware.RequireSelfUser).Put("/users/{userID}", handler.UpdateUser)
	router.With(middleware.RequireSelfUser).Get("/subscriptions/users/{userID}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	token, err := middleware.CreateToken(repo.user.Username, repo.user.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	send := func(method, path, body, cookie string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: "token", Value: cookie})
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}
	path := "/users/" + repo.user.ID.String()
	if res := send("PUT", path, `{"phonenumber":"628123456789","email":"new@example.test"}`, token); res.Code != 200 || repo.user.PhoneNumber != "628123456789" || repo.user.Email == nil || *repo.user.Email != "new@example.test" {
		t.Fatalf("update failed: %d %s", res.Code, res.Body.String())
	}
	if res := send("PUT", path, `{"password":"new password"}`, token); res.Code != 200 || bcrypt.CompareHashAndPassword([]byte(repo.user.Password), []byte("new password")) != nil || strings.Contains(res.Body.String(), "password") {
		t.Fatal("password update not hashed/private")
	}
	if res := send("PUT", path, `{"email":""}`, token); res.Code != 200 || repo.user.Email != nil {
		t.Fatal("email did not clear")
	}
	before := repo.updates
	for _, scenario := range []struct {
		path, body, token string
		status            int
	}{{path, `{}`, token, 400}, {path, `{"username":"Other"}`, token, 400}, {"/users/" + uuid.NewString(), `{"password":"attack"}`, token, 403}, {path, `{"email":"new@example.test"}`, "", 401}} {
		if res := send("PUT", scenario.path, scenario.body, scenario.token); res.Code != scenario.status {
			t.Fatalf("status: %d want %d", res.Code, scenario.status)
		}
	}
	if repo.updates != before {
		t.Fatal("rejected request reached repository")
	}
	if res := send("GET", "/subscriptions/users/"+uuid.NewString(), "", token); res.Code != 403 {
		t.Fatal("other user's subscription exposed")
	}
	repo.failure = errors.New("private update failure")
	res := send("PUT", path, `{"email":"valid@example.test"}`, token)
	if res.Code != 500 || strings.Contains(res.Body.String(), "private update") {
		t.Fatalf("error not masked: %s", res.Body.String())
	}
}
