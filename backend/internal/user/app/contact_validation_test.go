package app

import (
	"QueueLite/internal/apperror"
	"QueueLite/internal/user/domain"
	"context"
	"errors"
	"github.com/google/uuid"
	"strings"
	"testing"
)

type validationRepo struct {
	UserRepo
	user   domain.User
	writes int
}

func (r *validationRepo) CreateUser(_ context.Context, u *domain.User) (*domain.User, error) {
	r.writes++
	r.user = *u
	return u, nil
}
func (r *validationRepo) GetUser(context.Context, uuid.UUID) (*domain.User, error) {
	copy := r.user
	return &copy, nil
}
func (r *validationRepo) UpdateUser(_ context.Context, u *domain.User) error {
	r.writes++
	r.user = *u
	return nil
}
func TestRegistrationRejectsInvalidContactsAndPasswordBeforeWrite(t *testing.T) {
	for _, scenario := range []struct{ name, phone, password, email, code string }{
		{"bad-phone", "abc", "valid password", "", "INVALID_PHONE_NUMBER"}, {"bad-email", "081234567890", "valid password", "bad", "INVALID_EMAIL"},
		{"blank-password", "081234567890", "   ", "", "PASSWORD_REQUIRED"}, {"bcrypt-bytes", "081234567890", strings.Repeat("é", 37), "", "PASSWORD_TOO_LONG"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repo := &validationRepo{}
			_, err := NewUserService(repo, nil).RegisterUser(context.Background(), domain.User{Username: "Ayu", PhoneNumber: scenario.phone, Password: scenario.password, Email: &scenario.email})
			var appErr *apperror.Error
			if !errors.As(err, &appErr) || appErr.Code != scenario.code || repo.writes != 0 {
				t.Fatalf("validation/write: %v writes=%d", err, repo.writes)
			}
		})
	}
}
func TestValidRegistrationPreservesUnicodeAndPhoneVariants(t *testing.T) {
	for _, phone := range []string{"081234567890", "+62 812-3456-7890", "6281234567890"} {
		t.Run(phone, func(t *testing.T) {
			repo := &validationRepo{}
			user, err := NewUserService(repo, nil).RegisterUser(context.Background(), domain.User{Username: "  Ayu O’Connor-李  ", PhoneNumber: phone, Password: "password"})
			if err != nil || user.Username != "Ayu O’Connor-李" || repo.writes != 1 || user.Password == "password" {
				t.Fatalf("valid input rejected: %#v %v", user, err)
			}
		})
	}
}
func TestProfileTamperingCannotStoreInvalidContactOrPassword(t *testing.T) {
	for _, input := range []domain.User{{PhoneNumber: "abc"}, {Email: stringPointer("bad")}, {Password: strings.Repeat("é", 37)}, {Password: "   "}} {
		repo := &validationRepo{user: domain.User{ID: uuid.New(), PhoneNumber: "081234567890"}}
		if err := NewUserService(repo, nil).UpdateUser(context.Background(), repo.user.ID, input); err == nil || repo.writes != 0 {
			t.Fatalf("invalid profile stored: %v writes=%d", err, repo.writes)
		}
	}
}
func stringPointer(value string) *string { return &value }
