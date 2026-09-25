package inbound

type CreateQueueRequest struct {
	BusinessID string `json:"businessId"`
	UserID     string `json:"userId,omitempty"`
	Name       string `json:"name"`
	Priority   bool   `json:"priority"`
}

type UpdateQueueRequest struct {
	Name     *string `json:"name"`
	State    *string `json:"state"`
	Priority *bool   `json:"priority"`
}

type UpdateQueueStateRequest struct {
	State string `json:"state"`
}
