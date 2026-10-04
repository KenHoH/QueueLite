package app

import (
	"context"
	"errors"
	"strings"

	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/apperror"
	subscriptionapp "QueueLite/internal/subscription/app"
	"QueueLite/internal/user/domain"
	"QueueLite/internal/validation"

	"github.com/google/uuid"
)

type UserService struct {
	repo                UserRepo
	subscriptionService *subscriptionapp.SubscriptionService
}

func NewUserService(repo UserRepo, subscriptionService *subscriptionapp.SubscriptionService) *UserService {
	service := &UserService{
		repo:                repo,
		subscriptionService: subscriptionService,
	}
	return service
}

func (s *UserService) RegisterUser(ctx context.Context, user domain.User) (*domain.User, error) {
	user.Username = strings.TrimSpace(user.Username)
	user.PhoneNumber = strings.TrimSpace(user.PhoneNumber)
	if user.Email != nil {
		email := strings.TrimSpace(*user.Email)
		user.Email = &email
	}

	if user.Username == "" {
		return nil, apperror.New(apperror.KindInvalid, "USERNAME_REQUIRED", "missing username")
	}
	if strings.TrimSpace(user.Password) == "" {
		return nil, apperror.New(apperror.KindInvalid, "PASSWORD_REQUIRED", "missing password")
	}
	if user.PhoneNumber == "" {
		return nil, apperror.New(apperror.KindInvalid, "PHONENUMBER_REQUIRED", "missing phonenumber")
	}
	if !validation.Phone(user.PhoneNumber) {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_PHONE_NUMBER", "invalid phone number")
	}
	if user.Email != nil && *user.Email != "" && !validation.Email(*user.Email) {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_EMAIL", "invalid email address")
	}
	if len([]byte(user.Password)) > 72 {
		return nil, apperror.New(apperror.KindInvalid, "PASSWORD_TOO_LONG", "password must be at most 72 bytes")
	}

	hashedPassword, err := HashPassword(user.Password)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "INTERNAL_SERVER", "hash password error", err)
	}
	user.Password = hashedPassword

	record, err := s.repo.CreateUser(ctx, &user)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "INTERNAL_SERVER_ERROR", "failed to create user", err)
	}

	// creating the default subscription
	if _, err := s.subscriptionService.CreateDefaultUserSubscriptionPlan(ctx, record.ID); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *UserService) LoginUser(ctx context.Context, user domain.User) (*model.User, error) {
	user.Username = strings.TrimSpace(user.Username)

	userRecord, err := s.repo.GetUserByName(ctx, user.Username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, apperror.Wrap(apperror.KindUnauthorized, "INVALID_CREDENTIALS", "invalid username or password", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "INTERNAL_SERVER_ERROR", "failed to get user", err)
	}

	if !CheckPassword(*userRecord.Password, user.Password) {
		return nil, apperror.Wrap(apperror.KindUnauthorized, "INVALID_CREDENTIALS", "invalid username or password", ErrInvalidPassword)
	}

	return userRecord, nil
}

func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.repo.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, apperror.Wrap(apperror.KindNotFound, "USER_NOT_FOUND", "user not found", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "INTERNAL_SERVER_ERROR", "failed to get user", err)
	}
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, input domain.User) error {
	user, err := s.repo.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "USER_NOT_FOUND", "user not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "INTERNAL_SERVER_ERROR", "failed to get user", err)
	}

	if strings.TrimSpace(input.PhoneNumber) != "" {
		if !validation.Phone(input.PhoneNumber) {
			return apperror.New(apperror.KindInvalid, "INVALID_PHONE_NUMBER", "invalid phone number")
		}
		user.PhoneNumber = strings.TrimSpace(input.PhoneNumber)
	}
	if input.Email != nil {
		email := strings.TrimSpace(*input.Email)
		if email == "" {
			user.Email = nil
		} else {
			if !validation.Email(email) {
				return apperror.New(apperror.KindInvalid, "INVALID_EMAIL", "invalid email address")
			}
			user.Email = &email
		}
	}
	if input.Password != "" {
		if strings.TrimSpace(input.Password) == "" {
			return apperror.New(apperror.KindInvalid, "PASSWORD_REQUIRED", "password required")
		}
		if len([]byte(input.Password)) > 72 {
			return apperror.New(apperror.KindInvalid, "PASSWORD_TOO_LONG", "password must be at most 72 bytes")
		}
		hashedPassword, err := HashPassword(input.Password)
		if err != nil {
			return apperror.Wrap(apperror.KindInternal, "INTERNAL_SERVER", "hash password error", err)
		}
		user.Password = hashedPassword
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "USER_NOT_FOUND", "user not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "UPDATE_ERROR", "update user info failed", err)
	}
	return nil
}
