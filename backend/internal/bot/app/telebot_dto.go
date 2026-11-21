package app

const (
	NextQueueNotificationMessage    string = "Your Queue is next"
	CurrentQueueNotificationMessage string = "Your Queue is being called, please go to"
)

type TelebotMSGRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type ReplyMarkup struct {
	Keyboard        [][]KeyboardButton `json:"keyboard"`
	OneTimeKeyboard bool               `json:"one_time_keyboard"`
	ResizeKeyboard  bool               `json:"resize_keyboard"`
}

type KeyboardButton struct {
	Text           string `json:"text"`
	RequestContact bool   `json:"request_contact"`
}

type TelebotPhonenumberRequest struct {
	ChatID      string      `json:"chat_id"`
	Text        string      `json:"text"`
	ReplyMarkup ReplyMarkup `json:"reply_markup"`
}

type TeleUserResponse struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

type TeleChatResponse struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	Title     string `json:"title,omitempty"`
}

type TeleContactResponse struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name,omitempty"`
	UserID      int64  `json:"user_id,omitempty"`
	VCard       string `json:"vcard,omitempty"`
}

type TeleMessageResponse struct {
	MessageID int64                `json:"message_id"`
	From      *TeleUserResponse    `json:"from"`
	Chat      *TeleChatResponse    `json:"chat"`
	Date      int64                `json:"date"`
	Text      string               `json:"text,omitempty"`
	Contact   *TeleContactResponse `json:"contact,omitempty"` // Use a pointer (*) here
}

type TeleResultUpdateStatusResponse struct {
	UpdateID int64               `json:"update_id"`
	Message  TeleMessageResponse `json:"message"`
}

type TeleUpdateStatusResponse struct {
	Ok     bool                             `json:"ok"`
	Result []TeleResultUpdateStatusResponse `json:"result"`
}
