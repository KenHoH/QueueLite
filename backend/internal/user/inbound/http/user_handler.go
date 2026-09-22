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
	return &UserHandlerImpl{
		s: s,
	}
}

func (u *UserHandlerImpl) Routes() http.Handler {
	router := chi.NewRouter()

	router.Post("/", u.RegisterUser)
	router.Post("/login", u.LoginUser)
	router.Get("/{userID}", u.GetUser)

	return router
}

func (u *UserHandlerImpl) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var request UserRegisterRequest

	if err := decodeJSON(r, &request); err != nil {
		httpadapter.WriteError(w, apperror.New(
			apperror.KindInvalid,
			"INVALID_FORMAT",
			"format is invalid",
		))
		return
	}

	request.Username = strings.TrimSpace(request.Username)
	request.PhoneNumber = strings.TrimSpace(request.PhoneNumber)
	request.Email = strings.TrimSpace(request.Email)

	if request.Username == "" {
		httpadapter.WriteError(w, apperror.New(
			apperror.KindInvalid,
			"USERNAME_REQUIRED",
			"Missing username",
		))
		return
	}

	if request.Password == "" {
		httpadapter.WriteError(w, apperror.New(
			apperror.KindInvalid,
			"PASSWORD_REQUIRED",
			"Missing password",
		))
		return
	}

	if request.PhoneNumber == "" {
		httpadapter.WriteError(w, apperror.New(
			apperror.KindInvalid,
			"PHONENUMBER_REQUIRED",
			"Missing phonenumber",
		))
		return
	}

	var email *string
	if request.Email != "" {
		email = &request.Email
	}

	user, err := u.s.RegisterUser(
		r.Context(),
		domain.User{
			Username:    request.Username,
			PhoneNumber: request.PhoneNumber,
			Password:    request.Password,
			Email:       email,
		},
	)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusCreated, NewUserResponse(user))
}

func (u *UserHandlerImpl) LoginUser(w http.ResponseWriter, r *http.Request) {
	var request UserLoginRequest

	if err := decodeJSON(r, &request); err != nil {
		httpadapter.WriteError(w, apperror.New(
			apperror.KindInvalid,
			"INVALID_FORMAT",
			"format is invalid",
		))
		return
	}

	request.Username = strings.TrimSpace(request.Username)

	if request.Username == "" {
		httpadapter.WriteError(w, apperror.New(
			apperror.KindInvalid,
			"USERNAME_REQUIRED",
			"Missing username",
		))
		return
	}

	if request.Password == "" {
		httpadapter.WriteError(w, apperror.New(
			apperror.KindInvalid,
			"PASSWORD_REQUIRED",
			"Missing password",
		))
		return
	}

	if err := u.s.LoginUser(
		r.Context(),
		domain.User{
			Username: request.Username,
			Password: request.Password,
		},
	); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "login successful",
	})
}

func (u *UserHandlerImpl) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpadapter.WriteError(w, apperror.New(
			apperror.KindInvalid,
			"INVALID_USER_ID",
			"invalid user id",
		))
		return
	}

	user, err := u.s.GetUser(r.Context(), id)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, NewUserResponse(user))
}

func (u *UserHandlerImpl) UpdateUserPhoneNumber(w http.ResponseWriter, r *http.Request) {
	httpadapter.WriteError(w, apperror.New(
		apperror.KindNotImplemented,
		"NOT_IMPLEMENTED",
		"not implemented",
	))
}

func (u *UserHandlerImpl) UpdateUserEmail(w http.ResponseWriter, r *http.Request) {
	httpadapter.WriteError(w, apperror.New(
		apperror.KindNotImplemented,
		"NOT_IMPLEMENTED",
		"not implemented",
	))
}

func (u *UserHandlerImpl) UpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	httpadapter.WriteError(w, apperror.New(
		apperror.KindNotImplemented,
		"NOT_IMPLEMENTED",
		"not implemented",
	))
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
