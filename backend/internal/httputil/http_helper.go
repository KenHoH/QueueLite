package httputil

import (
	"encoding/json"
	"net/http"

	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
)

func WriteInvalid(w http.ResponseWriter, code string, message string) {
	httpadapter.WriteError(w, apperror.New(apperror.KindInvalid, code, message))
}

func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
