package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"QueueLite/internal/config"
	userapp "QueueLite/internal/user/app"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type TelebotService struct {
	userRepo   userapp.UserRepo
	rdb        *redis.Client
	httpClient *http.Client
}

const (
	TelebotBaseURL      string = "https://api.telegram.org/bot"
	UpdateEndpoint      string = "getUpdates"
	SendMessageEndpoint string = "sendMessage"
)

var ErrTelebotNotConfigured = errors.New("telebot api key is not configured")

func loadTelebotAPIKey() (string, error) {
	apiKey := strings.TrimSpace(config.LoadTelebotAPIKey())
	if apiKey == "" {
		return "", ErrTelebotNotConfigured
	}
	return apiKey, nil
}

func NewTelebotService(repo userapp.UserRepo, rdb ...*redis.Client) *TelebotService {
	var client *redis.Client
	if len(rdb) > 0 {
		client = rdb[0]
	}
	return &TelebotService{
		userRepo: repo,
		rdb:      client,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *TelebotService) client() *http.Client {
	if s != nil && s.httpClient != nil {
		return s.httpClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (s *TelebotService) postTelegramJSON(ctx context.Context, apiKey string, endpoint string, payload any) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode telegram payload: %w", err)
	}

	fullURL := fmt.Sprintf("%s%s/%s", TelebotBaseURL, apiKey, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client().Do(req)
	if err != nil {
		return fmt.Errorf("send telegram request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("read telegram response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("telegram api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (s *TelebotService) requestPhonenumber(ctx context.Context, chatID string) error {
	apiKey, err := loadTelebotAPIKey()
	if err != nil {
		return err
	}

	telebotMSGRequestData := TelebotPhonenumberRequest{
		ChatID: chatID,
		Text:   "Please share your phone number to enable notification",
		ReplyMarkup: ReplyMarkup{
			OneTimeKeyboard: true,
			ResizeKeyboard:  true,
			Keyboard: [][]KeyboardButton{
				{
					{
						Text:           "Share Phone Number",
						RequestContact: true,
					},
				},
			},
		},
	}
	return s.postTelegramJSON(ctx, apiKey, SendMessageEndpoint, telebotMSGRequestData)
}

func (s *TelebotService) SendMessageToUser(ctx context.Context, userID uuid.UUID, text string) error {
	if s == nil || s.userRepo == nil || userID == uuid.Nil {
		return nil
	}
	apiKey, err := loadTelebotAPIKey()
	if err != nil {
		if errors.Is(err, ErrTelebotNotConfigured) {
			return nil
		}
		return err
	}

	user, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, userapp.ErrUserNotFound) {
			// Guest tickets use ephemeral UUIDs that do not exist in the users table.
			return nil
		}
		return err
	}
	if user == nil || user.ChatID == nil || strings.TrimSpace(*user.ChatID) == "" {
		return nil
	}

	return s.postTelegramJSON(ctx, apiKey, SendMessageEndpoint, TelebotMSGRequest{
		ChatID: strings.TrimSpace(*user.ChatID),
		Text:   text,
	})
}

func (s *TelebotService) fetchTelegramUpdates(ctx context.Context, apiKey string) (*TeleUpdateStatusResponse, error) {
	fullURL := fmt.Sprintf("%s%s/%s", TelebotBaseURL, apiKey, UpdateEndpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create telegram getUpdates request: %w", err)
	}

	resp, err := s.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("send telegram getUpdates request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read telegram getUpdates response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("telegram getUpdates returned status %d", resp.StatusCode)
	}

	var apiResponse TeleUpdateStatusResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("unmarshal telegram updates: %w", err)
	}
	if !apiResponse.Ok {
		return nil, errors.New("telegram api returned ok=false")
	}
	return &apiResponse, nil
}

func (s *TelebotService) handleTelegramUpdate(ctx context.Context, update TeleResultUpdateStatusResponse) {
	msg := update.Message
	if msg.Chat == nil || msg.From == nil {
		return
	}

	chatID := strconv.FormatInt(msg.Chat.ID, 10)
	fromID := strconv.FormatInt(msg.From.ID, 10)
	if chatID != fromID {
		return
	}

	if msg.Contact != nil {
		if err := s.userRepo.InsertChatIDByPhoneNumber(ctx, chatID, msg.Contact.PhoneNumber); err != nil {
			log.Printf("[TELEBOT WARN] failed to link chat_id for update %d: %v", update.UpdateID, err)
		}
		return
	}
	if err := s.EnqueuePhoneRequest(ctx, chatID); err != nil {
		log.Printf("[TELEBOT WARN] failed to enqueue phone request for update %d: %v", update.UpdateID, err)
	}
}

func (s *TelebotService) GetUpdateStatus(ctx context.Context) error {
	apiKey, err := loadTelebotAPIKey()
	if err != nil {
		return err
	}

	apiResponse, err := s.fetchTelegramUpdates(ctx, apiKey)
	if err != nil {
		return err
	}
	for _, update := range apiResponse.Result {
		s.handleTelegramUpdate(ctx, update)
	}
	return nil
}
