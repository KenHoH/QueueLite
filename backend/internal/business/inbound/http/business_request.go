package inbound

type CreateBusinessRequest struct {
	Name        string  `json:"name"`
	Location    string  `json:"location"`
	Description *string `json:"description"`
	OpenTime    string  `json:"openTime"`
	CloseTime   string  `json:"closeTime"`
	Email       string  `json:"email"`
	PhoneNumber string  `json:"phoneNumber"`
}

type UpdateBusinessRequest struct {
	Name        *string `json:"name"`
	Location    *string `json:"location"`
	Description *string `json:"description"`
	OpenTime    *string `json:"openTime"`
	CloseTime   *string `json:"closeTime"`
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phoneNumber"`
}
