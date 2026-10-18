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
	"time"

	"QueueLite/internal/config"
	userapp "QueueLite/internal/user/app"

	"github.com/google/uuid"
)

type TelebotService struct {
	userRepo userapp.UserRepo
}

const (
	TelebotBaseURL      string = "https://api.telegram.org/bot"
	UpdateEndpoint      string = "getUpdates"
	SendMessageEndpoint string = "sendMessage"
)

func loadTelebotAPIKey() (string, error) {
	cfg, err := config.Load()
	if err != nil || cfg == nil || cfg.SecretKey == "" {
		return "", errors.New("invalid telebot api key")
	}
	return cfg.TelebotAPI, nil
}

func NewTelebotService(repo userapp.UserRepo) *TelebotService {
	return &TelebotService{
		userRepo: repo,
	}
}

func (s *TelebotService) requestPhonenumber(chatID string) error {
	apiKey, err := loadTelebotAPIKey()
	if err != nil {
		return err
	}
	fullURL := fmt.Sprintf("%s%s/%s", TelebotBaseURL, apiKey, SendMessageEndpoint)
	fmt.Println(fullURL)
	client := &http.Client{
		Timeout: 10 * time.Second,
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
	// 2. Marshal the struct into a JSON byte slice
	jsonBytes, err := json.Marshal(telebotMSGRequestData)
	if err != nil {
		log.Fatalf("Error encoding JSON: %v", err)
		return err
	}

	// 3. Create the POST request using bytes.NewBuffer to create an io.Reader
	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
		return err
	}

	// 4. Set headers (Important for APIs expecting JSON)
	req.Header.Set("Content-Type", "application/json")

	// 5. Execute the request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	// 6. Read and display the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
		return err
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response Body:\n%s\n", string(body))
	return nil
}

func (s *TelebotService) SendMessageToUser(ctx context.Context, userID uuid.UUID, text string) error {
	user, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	chatID := user.ChatID

	apiKey, err := loadTelebotAPIKey()
	if err != nil {
		return err
	}
	fullURL := fmt.Sprintf("%s%s/%s", TelebotBaseURL, apiKey, SendMessageEndpoint)
	fmt.Println(fullURL)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	telebotMSGRequestData := TelebotMSGRequest{
		ChatID: *chatID,
		Text:   text,
	}

	// 2. Marshal the struct into a JSON byte slice
	jsonBytes, err := json.Marshal(telebotMSGRequestData)
	if err != nil {
		log.Fatalf("Error encoding JSON: %v", err)
		return err
	}

	// 3. Create the POST request using bytes.NewBuffer to create an io.Reader
	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
		return err
	}

	// 4. Set headers (Important for APIs expecting JSON)
	req.Header.Set("Content-Type", "application/json")

	// 5. Execute the request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	// 6. Read and display the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
		return err
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response Body:\n%s\n", string(body))

	return nil
}

func (s *TelebotService) GetUpdateStatus(ctx context.Context) error {
	apiKey, err := loadTelebotAPIKey()
	if err != nil {
		return err
	}

	fullURL := fmt.Sprintf("%s%s/%s", TelebotBaseURL, apiKey, UpdateEndpoint)
	fmt.Println(fullURL)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
		return err
	}

	// 3. Execute the request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
		return err
	}

	// CRITICAL: Always defer closing the response body to prevent resource leaks
	defer resp.Body.Close()

	// 4. Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
		return err
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response Body:\n%s\n", string(body))

	var apiResponse TeleUpdateStatusResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		fmt.Printf("Error unmarshaling JSON: %v\n", err)
		return err
	}

	if !apiResponse.Ok {
		fmt.Println("Telegram returned ok: false")
		return fmt.Errorf("There's something wrong with telegram response")
	}

	for _, update := range apiResponse.Result {
		fmt.Printf("Processing Update ID: %d\n", update.UpdateID)

		msg := update.Message

		userID := strconv.FormatInt(msg.Chat.ID, 10)
		fromID := strconv.FormatInt(msg.From.ID, 10)
		if userID != fromID {
			return fmt.Errorf("Different ID missmatch")
		}

		// The phone number check
		if msg.Contact != nil {
			// insert to database
			phoneNumber := msg.Contact.PhoneNumber
			s.userRepo.InsertChatIDByPhoneNumber(ctx, userID, phoneNumber)
		} else {

			// TODO: implement message queue use go routine to create a new thread
			// Send Request Phone Number
			err := s.requestPhonenumber(userID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
