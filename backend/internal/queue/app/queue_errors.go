package app

import "errors"

var (
	ErrQueueAlreadyExists  = errors.New("queue already exists")
	ErrQueueNotFound       = errors.New("queue not found")
	ErrQueueNotImplemented = errors.New("queue not implemented")
)
