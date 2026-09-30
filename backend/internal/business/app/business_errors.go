package app

import (
	"QueueLite/internal/business/domain"
	"errors"
)

var (
	ErrBusinessAlreadyExists  = errors.New("business already exists")
	ErrBusinessNotFound       = domain.ErrBusinessNotFound
	ErrBusinessNotImplemented = errors.New("business not implemented")
)
