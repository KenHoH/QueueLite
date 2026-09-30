package inbound

type CreateQueueRequest struct {
	BusinessID  string `json:"businessId"`
	Username    string `json:"username"`
	PhoneNumber string `json:"phoneNumber"`
}

type CustomerJoinRequest struct {
	Username    string `json:"username"`
	PhoneNumber string `json:"phoneNumber"`
}

type RegisterQueueByQRRequest = CustomerJoinRequest

type UpdateQueueRequest struct {
	Name     *string `json:"name"`
	State    *string `json:"state"`
	Priority *bool   `json:"priority"`
}

type UpdateQueueStateRequest struct {
	State string `json:"state"`
}
