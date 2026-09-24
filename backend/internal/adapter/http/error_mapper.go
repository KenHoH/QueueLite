package httpadapter

import (
	"QueueLite/internal/apperror"
	"errors"
	"net/http"
)

func statusFromError(err error) int {
	var appErr *apperror.Error

	if !errors.As(err, &appErr) {
		return http.StatusInternalServerError
	}

	switch appErr.Kind {

	case apperror.KindInvalid:
		return http.StatusBadRequest

	case apperror.KindNotFound:
		return http.StatusNotFound

	case apperror.KindConflict:
		return http.StatusConflict

	case apperror.KindUnauthorized:
		return http.StatusUnauthorized

	case apperror.KindForbidden:
		return http.StatusForbidden

	case apperror.KindNotImplemented:
		return http.StatusNotImplemented

	default:
		return http.StatusInternalServerError
	}
}
