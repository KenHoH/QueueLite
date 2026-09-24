package httpadapter

import (
	"QueueLite/internal/apperror"
	"errors"
	"net/http"
)

func WriteError(w http.ResponseWriter, err error) {
	var appErr *apperror.Error

	if !errors.As(err, &appErr) {
		WriteJSON(
			w,
			http.StatusInternalServerError,
			ErrorResponse{
				Error: ErrorDetail{
					Code:    "INTERNAL_ERROR",
					Message: "internal server error",
				},
			},
		)

		return
	}

	status := statusFromError(appErr)

	message := appErr.Message

	if appErr.Kind == apperror.KindInternal {
		message = "internal server error"
	}

	WriteJSON(
		w,
		status,
		ErrorResponse{
			Error: ErrorDetail{
				Code:    appErr.Code,
				Message: message,
			},
		},
	)
}
