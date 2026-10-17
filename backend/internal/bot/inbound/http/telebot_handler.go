package http

import (
	"net/http"

	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/bot/app"
)

type TelebotHandler struct {
	s *app.TelebotService
}

func NewTelebotHandler(service *app.TelebotService) *TelebotHandler {
	return &TelebotHandler{
		s: service,
	}
}

func (t *TelebotHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	err := t.s.GetUpdateStatus(r.Context())
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "update status successful"})
}

// Testing Purpose
// func (t *TelebotHandler) NotifyUser(w http.ResponseWriter, r *http.Request) {
// 	err := t.s.SendMessageToUser("", "")
// 	if err != nil {
// 		httpadapter.WriteError(w, err)
// 		return
// 	}
// 	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "update status successful"})
// }

// func (t *TelebotHandler) RequestPhonenumer(w http.ResponseWriter, r *http.Request) {
// 	err := t.s.RequestPhonenumber("6019875662")
// 	if err != nil {
// 		httpadapter.WriteError(w, err)
// 		return
// 	}
// 	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "update status successful"})
// }
