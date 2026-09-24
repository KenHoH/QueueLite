package app

import "errors"

var (
	ErrCounterAlreadyExists  = errors.New("counter already exists")
	ErrCounterNotFound       = errors.New("counter not found")
	ErrCounterNotImplemented = errors.New("counter not implemented")
)
