package app

import "errors"

var (
	ErrBusinessAlreadyExists  = errors.New("business already exists")
	ErrBusinessNotFound       = errors.New("business not found")
	ErrBusinessNotImplemented = errors.New("business not implemented")
)
