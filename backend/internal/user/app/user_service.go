package app

import (
	"QueueLite/internal/apperror"
	"QueueLite/internal/user/domain"
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo UserRepo
}

func NewUserService(repo UserRepo) *UserService {
	return &UserService{
		repo: repo,
	}
}

func checkPassword(hashedPassword string, password string) bool {
	byteHashedPassword := []byte(hashedPassword)
	bytePassword := []byte(password)

	err := bcrypt.CompareHashAndPassword(byteHashedPassword, bytePassword)
	if err != nil {
		return false
	}
	return true
}
func (s *UserService) RegisterUser(ctx context.Context, user domain.User) (*domain.User, error) {
	bytePassword := []byte(user.Password)
	hashedPassword, err := bcrypt.GenerateFromPassword(bytePassword, bcrypt.DefaultCost)

	if err != nil {
		return nil, apperror.Wrap(
			apperror.KindInternal,
			"INTERNAL_SERVER",
			"Hash Password Error",
			err,
		)
	}
	user.Password = string(hashedPassword)
	record, err := s.repo.CreateUser(ctx, &user)
	if err != nil {
		return nil, apperror.Wrap(
			apperror.KindInternal,
			"INTERNAL_SERVER_ERROR",
			"Failed to create user",
			err,
		)
	}
	return record, nil
}
func (s *UserService) LoginUser(ctx context.Context, user domain.User) error {
	userRecord, err := s.repo.GetUserByName(ctx, user.Username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return apperror.Wrap(
				apperror.KindUnauthorized,
				"INVALID_CREDENTIALS",
				"invalid username or password",
				err,
			)
		}

		return apperror.Wrap(
			apperror.KindInternal,
			"INTERNAL_SERVER_ERROR",
			"failed to get user",
			err,
		)
	}

	if !checkPassword(userRecord.Password, user.Password) {
		return apperror.Wrap(
			apperror.KindUnauthorized,
			"INVALID_CREDENTIALS",
			"invalid username or password",
			ErrInvalidPassword,
		)
	}

	return nil
}

func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.repo.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, apperror.Wrap(
				apperror.KindNotFound,
				"USER_NOT_FOUND",
				"user not found",
				err,
			)
		}

		return nil, apperror.Wrap(
			apperror.KindInternal,
			"INTERNAL_SERVER_ERROR",
			"failed to get user",
			err,
		)
	}

	return user, nil
}
func (s *UserService) UpdateUserInfo(ctx context.Context, user domain.User) error {
	err := s.repo.UpdateUser(ctx, &user)
	if err != nil {
		return apperror.Wrap(
			apperror.KindInternal,
			"UPDATE_ERROR",
			"Update Phone Number Error",
			err,
		)
	}

	return nil
}
