package app

type TelebotMSGRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}
