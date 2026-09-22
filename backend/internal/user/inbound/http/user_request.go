package inbound

type UserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type UserRegisterRequest struct {
	Username    string `json:"username"`
	PhoneNumber string `json:"phonenumber"`
	Password    string `json:"password"`
	Email       string `json:"email"`
}

type UpdateUserInformationRequest struct {
	UserID      string `json:"id"`
	PhoneNumber string `json:"phonenumber"`
	Password    string `json:"password"`
	Email       string `json:"email"`
}
