package inbound

import (
	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	"QueueLite/internal/user/app"
	"QueueLite/internal/user/domain"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type UserHandlerImpl struct {
	s *app.UserService
}

func NewUserHandler(s *app.UserService) *UserHandlerImpl {
	return &UserHandlerImpl{s: s}
}

func (u *UserHandlerImpl) Routes() http.Handler {
	router := chi.NewRouter()

	router.Post("/", u.RegisterUser)
	router.Post("/login", u.LoginUser)
	router.Get("/{userID}", u.GetUser)
	router.Put("/{userID}", u.UpdateUser)

	return router
}

func (u *UserHandlerImpl) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var request UserRegisterRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	request.Username = strings.TrimSpace(request.Username)
	request.PhoneNumber = strings.TrimSpace(request.PhoneNumber)
	request.Email = strings.TrimSpace(request.Email)

	if request.Username == "" {
		writeInvalid(w, "USERNAME_REQUIRED", "Missing username")
		return
	}
	if request.Password == "" {
		writeInvalid(w, "PASSWORD_REQUIRED", "Missing password")
		return
	}
	if request.PhoneNumber == "" {
		writeInvalid(w, "PHONENUMBER_REQUIRED", "Missing phonenumber")
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
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	request.Username = strings.TrimSpace(request.Username)
	if request.Username == "" {
		writeInvalid(w, "USERNAME_REQUIRED", "Missing username")
		return
	}
	if request.Password == "" {
		writeInvalid(w, "PASSWORD_REQUIRED", "Missing password")
		return
	}

	if err := u.s.LoginUser(r.Context(), domain.User{Username: request.Username, Password: request.Password}); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "login successful"})
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
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	if request.PhoneNumber == nil && request.Password == nil && request.Email == nil {
		writeInvalid(w, "NO_USER_FIELDS", "no user fields provided")
		return
	}

	input := domain.User{}
	if request.PhoneNumber != nil {
		input.PhoneNumber = *request.PhoneNumber
	}
	if request.Password != nil {
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
		writeInvalid(w, code, message)
		return uuid.Nil, false
	}
	return id, true
}

func writeInvalid(w http.ResponseWriter, code string, message string) {
	httpadapter.WriteError(w, apperror.New(apperror.KindInvalid, code, message))
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
