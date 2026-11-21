package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	userapp "QueueLite/internal/user/app"
	userdomain "QueueLite/internal/user/domain"

	"github.com/google/uuid"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type stubUserRepo struct {
	userapp.UserRepo
	user *userdomain.User
}

func (r *stubUserRepo) GetUser(_ context.Context, id uuid.UUID) (*userdomain.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, userapp.ErrUserNotFound
	}
	return r.user, nil
}

func TestSendMessageToUserHandlesMissingConfigNilChatIDAndGuestSafely(t *testing.T) {
	uid := uuid.New()
	svc := NewTelebotService(&stubUserRepo{
		user: &userdomain.User{ID: uid, Username: "Ayu", PhoneNumber: "6281234567890", ChatID: nil},
	})

	// Missing TELEBOT_API_KEY should gracefully no-op instead of erroring or panicking.
	t.Setenv("TELEBOT_API_KEY", "")
	if err := svc.SendMessageToUser(context.Background(), uid, "hello"); err != nil {
		t.Fatalf("expected nil when api key missing, got %v", err)
	}

	// Nil ChatID should no-op safely without panicking even when api key is set.
	t.Setenv("TELEBOT_API_KEY", "test-key")
	if err := svc.SendMessageToUser(context.Background(), uid, "hello"); err != nil {
		t.Fatalf("expected nil when user has no chat id, got %v", err)
	}

	// Unknown guest UUID should no-op safely.
	if err := svc.SendMessageToUser(context.Background(), uuid.New(), "hello"); err != nil {
		t.Fatalf("expected nil for guest uuid, got %v", err)
	}
}

func TestPostTelegramJSONReturnsErrorInsteadOfFatalOnFailure(t *testing.T) {
	var hits atomic.Int32
	chatID := "123456789"
	uid := uuid.New()
	svc := NewTelebotService(&stubUserRepo{
		user: &userdomain.User{ID: uid, Username: "Ayu", PhoneNumber: "6281234567890", ChatID: &chatID},
	})
	svc.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			hits.Add(1)
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(bytes.NewBufferString("telegram unavailable")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	t.Setenv("TELEBOT_API_KEY", "test-key")
	err := svc.SendMessageToUser(context.Background(), uid, "Your turn")
	if err == nil {
		t.Fatal("expected error on 502 telegram response")
	}
	if hits.Load() != 1 {
		t.Fatalf("expected 1 telegram request attempt, got %d", hits.Load())
	}
}
