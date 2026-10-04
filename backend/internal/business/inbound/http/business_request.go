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

type UpsertBusinessMemberRequest struct {
	Identifier string `json:"identifier"`
	Role       string `json:"role"`
}

type UpdateBusinessRequest struct {
	Operational *bool   `json:"operational"`
	Name        *string `json:"name"`
	Location    *string `json:"location"`
	Description *string `json:"description"`
	OpenTime    *string `json:"openTime"`
	CloseTime   *string `json:"closeTime"`
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phoneNumber"`
}
