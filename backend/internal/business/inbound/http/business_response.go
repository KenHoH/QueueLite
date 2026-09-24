package inbound

import "QueueLite/internal/business/domain"

type BusinessResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Location    string  `json:"location"`
	Description *string `json:"description,omitempty"`
	Operational bool    `json:"operational"`
	OpenTime    string  `json:"openTime"`
	CloseTime   string  `json:"closeTime"`
	Email       string  `json:"email"`
	PhoneNumber string  `json:"phoneNumber"`
}

func NewBusinessResponse(business *domain.Business) BusinessResponse {
	return BusinessResponse{
		ID:          business.ID.String(),
		Name:        business.Name,
		Location:    business.Location,
		Description: business.Description,
		Operational: business.Operational,
		OpenTime:    business.OpenTime.Format("15:04"),
		CloseTime:   business.CloseTime.Format("15:04"),
		Email:       business.Email,
		PhoneNumber: business.PhoneNumber,
	}
}

func NewBusinessResponses(businesses []domain.Business) []BusinessResponse {
	responses := make([]BusinessResponse, 0, len(businesses))
	for i := range businesses {
		responses = append(responses, NewBusinessResponse(&businesses[i]))
	}
	return responses
}
