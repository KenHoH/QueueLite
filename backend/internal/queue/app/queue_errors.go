package app

import (
	"errors"

	"QueueLite/internal/apperror"
)

var (
	ErrQueueAlreadyExists  = errors.New("queue already exists")
	ErrQueueNotFound       = errors.New("queue not found")
	ErrQueueNotImplemented = errors.New("queue not implemented")
)

func CustomerQuotaError(err error) error {
	var appErr *apperror.Error
	if errors.As(err, &appErr) && (appErr.Code == "QUEUE_FULL" || appErr.Code == "BUSINESS_QUEUE_FULL") {
		return apperror.New(apperror.KindConflict, "BUSINESS_QUEUE_FULL", "the business queue is full")
	}
	return apperror.Wrap(apperror.KindInternal, "BUSINESS_QUOTA_ERROR", "failed to check business capacity", err)
}
