package app

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
