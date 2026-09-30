package inbound

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/middleware"
	"QueueLite/internal/user/app"
	"QueueLite/internal/user/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type authRepo struct {
	app.UserRepo
	user domain.User
}

func (repo authRepo) GetUser(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if id != repo.user.ID {
		return nil, app.ErrUserNotFound
	}
	return &repo.user, nil
}
func (repo authRepo) GetUserByName(_ context.Context, _ string) (*model.User, error) {
	return nil, app.ErrUserNotFound
}

func TestCurrentUser(t *testing.T) {
	t.Setenv("DATABASE_URL", "test")
	t.Setenv("SECRET", "test-secret")
	t.Setenv("REDIS_HOST", "test")
	repo := authRepo{user: domain.User{ID: uuid.New(), Username: "Matthew", PhoneNumber: "08", Password: "never-return"}}
	handler := NewUserHandler(app.NewUserService(repo, nil))
	router := chi.NewRouter()
	router.With(middleware.AuthMiddleware(repo)).Get("/users/me", handler.GetCurrentUser)
	token, err := middleware.CreateToken(repo.user.Username, repo.user.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, token string
		status      int
	}{
		{"authenticated", token, 200}, {"absent", "", 401}, {"invalid", "invalid", 401},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/users/me?userID="+uuid.NewString(), nil)
			if test.token != "" {
				request.AddCookie(&http.Cookie{Name: "token", Value: test.token})
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status %d", response.Code)
			}
			if test.status == 200 {
				var body map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["id"] != repo.user.ID.String() || body["username"] != "Matthew" || body["phonenumber"] != "08" {
					t.Fatalf("unexpected user: %v", body)
				}
				if _, exists := body["password"]; exists {
					t.Fatal("password exposed")
				}
			}
		})
	}
	response := httptest.NewRecorder()
	handler.GetCurrentUser(response, httptest.NewRequest(http.MethodGet, "/users/me", nil))
	if response.Code != 401 {
		t.Fatal("missing context must be unauthorized")
	}
}

func TestLogout(t *testing.T) {
	handler := NewUserHandler(nil)
	for _, cookie := range []string{"", "token=old; queueToken=queue; guestToken=guest"} {
		request := httptest.NewRequest(http.MethodPost, "/users/logout", nil)
		request.Header.Set("Cookie", cookie)
		response := httptest.NewRecorder()
		handler.LogoutUser(response, request)
		if response.Code != 200 {
			t.Fatalf("status %d", response.Code)
		}
		cookies := response.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatal("must change only one cookie")
		}
		cleared := cookies[0]
		if cleared.Name != "token" || cleared.Value != "" || cleared.Path != "/" || !cleared.HttpOnly || cleared.SameSite != http.SameSiteLaxMode || cleared.MaxAge >= 0 || !cleared.Expires.Before(time.Now()) || cleared.Secure {
			t.Fatalf("incorrect cookie: %v", cleared)
		}
		var body map[string]string
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["message"] != "logout successful" {
			t.Fatalf("incorrect response: %s", response.Body.String())
		}
	}
}
