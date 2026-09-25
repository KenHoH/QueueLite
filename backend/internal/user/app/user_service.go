package app

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/apperror"
	subscriptionapp "QueueLite/internal/subscription/app"
	"QueueLite/internal/user/domain"
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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

func checkPassword(hashedPassword string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
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
	if user.Password == "" {
		return nil, apperror.New(apperror.KindInvalid, "PASSWORD_REQUIRED", "missing password")
	}
	if user.PhoneNumber == "" {
		return nil, apperror.New(apperror.KindInvalid, "PHONENUMBER_REQUIRED", "missing phonenumber")
	}

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "INTERNAL_SERVER", "hash password error", err)
	}
	user.Password = hashedPassword

	record, err := s.repo.CreateUser(ctx, &user)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "INTERNAL_SERVER_ERROR", "failed to create user", err)
	}
	if s.subscriptionService != nil {
		if _, err := s.subscriptionService.CreateDefaultUserSubscriptionPlan(ctx, record.ID); err != nil {
			return nil, err
		}
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

	if !checkPassword(*userRecord.Password, user.Password) {
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
		user.PhoneNumber = strings.TrimSpace(input.PhoneNumber)
	}
	if input.Email != nil {
		email := strings.TrimSpace(*input.Email)
		if email == "" {
			user.Email = nil
		} else {
			user.Email = &email
		}
	}
	if input.Password != "" {
		hashedPassword, err := hashPassword(input.Password)
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
