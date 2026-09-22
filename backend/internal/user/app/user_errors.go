package app

import "errors"

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidEmail      = errors.New("invalid email")
	ErrInvalidPassword   = errors.New("invalid password")
)
