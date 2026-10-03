package inbound

import (
	"net/http"
	"strings"
	"time"

	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/middleware"
	"QueueLite/internal/user/app"
	"QueueLite/internal/user/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type UserHandlerImpl struct {
	s *app.UserService
}

func NewUserHandler(s *app.UserService) *UserHandlerImpl {
	return &UserHandlerImpl{s: s}
}

func (u *UserHandlerImpl) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var request UserRegisterRequest
	if err := DecodeJSON(r, &request); err != nil {
		WriteInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	request.Username = strings.TrimSpace(request.Username)
	request.PhoneNumber = strings.TrimSpace(request.PhoneNumber)
	request.Email = strings.TrimSpace(request.Email)

	if request.Username == "" {
		WriteInvalid(w, "USERNAME_REQUIRED", "Missing username")
		return
	}
	if request.Password == "" {
		WriteInvalid(w, "PASSWORD_REQUIRED", "Missing password")
		return
	}
	if request.PhoneNumber == "" {
		WriteInvalid(w, "PHONENUMBER_REQUIRED", "Missing phonenumber")
		return
	}

	var email *string
	if request.Email != "" {
		email = &request.Email
	}

	user, err := u.s.RegisterUser(r.Context(), domain.User{
		Username:    request.Username,
		PhoneNumber: request.PhoneNumber,
		Password:    request.Password,
		Email:       email,
	})
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusCreated, NewUserResponse(user))
}

func (u *UserHandlerImpl) LoginUser(w http.ResponseWriter, r *http.Request) {
	var request UserLoginRequest
	if err := DecodeJSON(r, &request); err != nil {
		WriteInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	request.Username = strings.TrimSpace(request.Username)
	if request.Username == "" {
		WriteInvalid(w, "USERNAME_REQUIRED", "Missing username")
		return
	}
	if request.Password == "" {
		WriteInvalid(w, "PASSWORD_REQUIRED", "Missing password")
		return
	}

	userEntity, err := u.s.LoginUser(r.Context(), domain.User{Username: request.Username, Password: request.Password})
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	userStringID := userEntity.ID.String()
	tokenString, err := middleware.CreateToken(userEntity.Username, userStringID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  time.Now().Add(time.Hour * 24),
		HttpOnly: true,                 // Prevents JavaScript from reading the cookie (Mitigates XSS)
		Secure:   false,                // Set to true in production to force HTTPS
		Path:     "/",                  // Accessible across the entire domain
		SameSite: http.SameSiteLaxMode, // Controls cross-site cookie behavior
	})

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "login successful"})
}

func (u *UserHandlerImpl) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIdFromContext(r.Context())
	id, err := uuid.Parse(userID)
	if !ok || err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := u.s.GetUser(r.Context(), id)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, NewUserResponse(user))
}

func (u *UserHandlerImpl) LogoutUser(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: "token", Value: "", Path: "/", HttpOnly: true,
		Secure: false, SameSite: http.SameSiteLaxMode,
		MaxAge: -1, Expires: time.Unix(1, 0),
	})
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "logout successful"})
}

func (u *UserHandlerImpl) GetUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "userID", "INVALID_USER_ID", "invalid user id")
	if !ok {
		return
	}

	user, err := u.s.GetUser(r.Context(), id)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, NewUserResponse(user))
}

func (u *UserHandlerImpl) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "userID", "INVALID_USER_ID", "invalid user id")
	if !ok {
		return
	}

	var request UpdateUserInformationRequest
	if err := DecodeJSON(r, &request); err != nil {
		WriteInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	if request.PhoneNumber == nil && request.Password == nil && request.Email == nil {
		WriteInvalid(w, "NO_USER_FIELDS", "no user fields provided")
		return
	}

	input := domain.User{}
	if request.PhoneNumber != nil {
		if strings.TrimSpace(*request.PhoneNumber) == "" {
			WriteInvalid(w, "PHONENUMBER_REQUIRED", "phone number required")
			return
		}
		input.PhoneNumber = *request.PhoneNumber
	}
	if request.Password != nil {
		if strings.TrimSpace(*request.Password) == "" {
			WriteInvalid(w, "PASSWORD_REQUIRED", "password required")
			return
		}
		input.Password = *request.Password
	}
	if request.Email != nil {
		input.Email = request.Email
	}

	if err := u.s.UpdateUser(r.Context(), id, input); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "user updated"})
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, name string, code string, message string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, name)))
	if err != nil {
		WriteInvalid(w, code, message)
		return uuid.Nil, false
	}
	return id, true
}
