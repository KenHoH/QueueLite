package inbound

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/middleware"
	"QueueLite/internal/queue/domain"

	"github.com/google/uuid"
)

func prepareSSE(w http.ResponseWriter) (http.Flusher, bool) {
	// Disable server WriteTimeout for long-lived SSE connections.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return nil, false
	}
	return flusher, true
}

// streamBusinessEvents sets up an SSE response, subscribes to a business's update channel,
// sends the initial frame, and streams updates with 25s keepalive heartbeats until send returns false or ctx ends.
func (h *QueueHandlerImpl) streamBusinessEvents(
	w http.ResponseWriter,
	r *http.Request,
	businessID uuid.UUID,
	send func(w http.ResponseWriter, flusher http.Flusher, initial bool) bool,
) {
	flusher, ok := prepareSSE(w)
	if !ok {
		return
	}

	pubsub := h.s.SubscribeQueueChannel(r.Context(), businessID.String())
	defer pubsub.Close()
	ch := pubsub.Channel()

	if !send(w, flusher, true) {
		return
	}

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case _, ok := <-ch:
			if !ok || !send(w, flusher, false) {
				return
			}
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data []byte) bool {
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

func (h *QueueHandlerImpl) SSEHandler(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}

	h.streamBusinessEvents(w, r, businessID, func(w http.ResponseWriter, flusher http.Flusher, _ bool) bool {
		response, err := h.s.GetWaitingQueueSnapshot(r.Context(), businessID)
		if err != nil {
			writeSSEEvent(w, flusher, "error", []byte(err.Error()))
			return false
		}
		payload, err := json.Marshal(NewQueueSnapshotResponses(response))
		if err != nil {
			writeSSEEvent(w, flusher, "error", []byte(err.Error()))
			return false
		}
		return writeSSEEvent(w, flusher, "queue.update", payload)
	})
}

// CustomerQueueSSEHandler streams real-time CustomerQueueStatusResponse updates for a single ticket.
func (h *QueueHandlerImpl) CustomerQueueSSEHandler(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	queue, hasPreloaded := middleware.LoadedQueueFromContext(r.Context())
	if !hasPreloaded || queue.ID != queueID {
		var err error
		queue, err = h.s.GetQueue(r.Context(), queueID)
		if err != nil {
			httpadapter.WriteError(w, err)
			return
		}
	}

	h.streamBusinessEvents(w, r, queue.BusinessID, func(w http.ResponseWriter, flusher http.Flusher, initial bool) bool {
		var preloaded *domain.Queue
		if initial {
			preloaded = queue
		}
		status, err := h.s.GetCustomerQueueStatusWithQueue(r.Context(), queueID, preloaded)
		if err != nil {
			writeSSEEvent(w, flusher, "error", []byte(`{"code":"STATUS_UNAVAILABLE"}`))
			return false
		}
		payload, err := json.Marshal(NewCustomerQueueStatusResponse(status))
		if err != nil {
			return false
		}
		if !writeSSEEvent(w, flusher, "status", payload) {
			return false
		}
		return !status.Queue.IsTerminal()
	})
}

// BusinessQueueSSEHandler notifies authorized staff dashboards/counters when business queue state changes.
func (h *QueueHandlerImpl) BusinessQueueSSEHandler(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}

	payload := []byte(fmt.Sprintf(`{"businessId":%q}`, businessID.String()))
	h.streamBusinessEvents(w, r, businessID, func(w http.ResponseWriter, flusher http.Flusher, initial bool) bool {
		event := "queue.update"
		if initial {
			event = "connected"
		}
		return writeSSEEvent(w, flusher, event, payload)
	})
}
