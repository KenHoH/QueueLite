package inbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	"QueueLite/internal/config"
	"QueueLite/internal/middleware"
	"QueueLite/internal/queue/app"
	"QueueLite/internal/queue/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type QueueHandlerImpl struct {
	s *app.QueueService
}

func NewQueueHandler(s *app.QueueService) *QueueHandlerImpl {
	return &QueueHandlerImpl{
		s: s,
	}
}

func resolvePublicBaseURL(r *http.Request) string {
	if configured := config.PublicBaseURL(); configured != "" {
		return configured
	}
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func (h *QueueHandlerImpl) GenerateQRCodeImage(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}
	fileBytes, err := h.s.GenerateQRImage(resolvePublicBaseURL(r), businessID.String())
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", `attachment; filename="qrcode.png"`)
	http.ServeContent(w, r, "qrcode.png", time.Now(), bytes.NewReader(fileBytes))
}

func attachActiveQueue(response *QueueQRResolveResponse, q *domain.Queue) {
	qIDStr := q.ID.String()
	qResp := NewQueueResponse(q)
	response.ActiveQueueID = &qIDStr
	response.ActiveQueue = &qResp
}

func (h *QueueHandlerImpl) ResolveQueueQR(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}

	if err := h.s.ValidateBusinessExists(r.Context(), businessID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	userID, authenticated := middleware.OptionalUserIdFromContext(r.Context())
	response := QueueQRResolveResponse{
		BusinessID:        businessID.String(),
		Authenticated:     authenticated,
		RequiresGuestForm: !authenticated,
	}
	if !authenticated {
		response.RequiredFields = []string{"username", "phoneNumber"}
		if queues, err := h.resolveGuestQueuesFromRequest(r); err == nil {
			for i := range queues {
				if queues[i].BusinessID == businessID && !queues[i].IsTerminal() {
					response.RequiresGuestForm = false
					response.RequiredFields = nil
					attachActiveQueue(&response, &queues[i])
					break
				}
			}
		}
	} else if parsedUser, err := uuid.Parse(userID); err == nil {
		active, err := h.s.GetActiveQueueByUserAndBusiness(r.Context(), parsedUser, businessID)
		if err == nil && active != nil && !active.IsTerminal() {
			attachActiveQueue(&response, active)
		}
	}

	httpadapter.WriteJSON(w, http.StatusOK, response)
}

type guestTokenData struct {
	guestID     string
	businessID  string
	phoneNumber string
}

func extractGuestRequestTokens(r *http.Request) ([]guestTokenData, []string) {
	var guestTokens []guestTokenData
	var queueTokenIDs []string

	for _, cookie := range r.Cookies() {
		if cookie.Name == "guestToken" || strings.HasPrefix(cookie.Name, "guestToken_") {
			guestID, bID, phone, err := middleware.ValidateGuestToken(cookie.Value)
			if err == nil && guestID != "" && bID != "" {
				guestTokens = append(guestTokens, guestTokenData{
					guestID:     guestID,
					businessID:  bID,
					phoneNumber: phone,
				})
			}
		}
		if cookie.Name == "queueToken" || strings.HasPrefix(cookie.Name, "queueToken_") {
			qID, err := middleware.ValidateQueueToken(cookie.Value)
			if err == nil && qID != "" {
				queueTokenIDs = append(queueTokenIDs, qID)
			}
		}
	}
	return guestTokens, queueTokenIDs
}

func (h *QueueHandlerImpl) guestOwnsAnyToken(ctx context.Context, queue *domain.Queue, guestTokens []guestTokenData) bool {
	businessID := queue.BusinessID.String()
	for _, gt := range guestTokens {
		if gt.businessID != businessID {
			continue
		}
		if owned, err := h.s.GuestOwnsQueue(ctx, queue, gt.guestID, gt.businessID, gt.phoneNumber); err == nil && owned {
			return true
		}
	}
	return false
}

func (h *QueueHandlerImpl) resolveOwnedTokenQueues(
	ctx context.Context,
	queueTokenIDs []string,
	guestTokens []guestTokenData,
	seen map[uuid.UUID]bool,
) []domain.Queue {
	var activeQueues []domain.Queue
	for _, qIDStr := range queueTokenIDs {
		qUUID, err := uuid.Parse(qIDStr)
		if err != nil || seen[qUUID] {
			continue
		}
		queue, err := h.s.GetQueue(ctx, qUUID)
		if err != nil || queue == nil || queue.IsTerminal() {
			continue
		}
		if h.guestOwnsAnyToken(ctx, queue, guestTokens) {
			seen[queue.ID] = true
			activeQueues = append(activeQueues, *queue)
		}
	}
	return activeQueues
}

func (h *QueueHandlerImpl) appendFallbackGuestQueues(
	ctx context.Context,
	guestTokens []guestTokenData,
	activeQueues []domain.Queue,
	seen map[uuid.UUID]bool,
) []domain.Queue {
	seenBusinesses := make(map[uuid.UUID]bool, len(activeQueues))
	for _, aq := range activeQueues {
		seenBusinesses[aq.BusinessID] = true
	}

	for _, gt := range guestTokens {
		bUUID, err := uuid.Parse(gt.businessID)
		if err != nil || seenBusinesses[bUUID] {
			continue
		}
		queue, err := h.s.GetActiveGuestQueue(ctx, gt.guestID, gt.businessID, gt.phoneNumber)
		if err == nil && queue != nil && !queue.IsTerminal() && !seen[queue.ID] {
			seen[queue.ID] = true
			seenBusinesses[bUUID] = true
			activeQueues = append(activeQueues, *queue)
		}
	}
	return activeQueues
}

func (h *QueueHandlerImpl) resolveGuestQueuesFromRequest(r *http.Request) ([]domain.Queue, error) {
	guestTokens, queueTokenIDs := extractGuestRequestTokens(r)
	seen := make(map[uuid.UUID]bool)

	// 1. Check queues identified by queueTokens.
	activeQueues := h.resolveOwnedTokenQueues(r.Context(), queueTokenIDs, guestTokens, seen)
	// 2. Also check businesses identified by guestTokens in case queueToken was scoped or absent.
	activeQueues = h.appendFallbackGuestQueues(r.Context(), guestTokens, activeQueues, seen)

	return activeQueues, nil
}

func (h *QueueHandlerImpl) JoinBusinessQueue(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}

	var request CustomerJoinRequest
	if _, authenticated := middleware.OptionalUserIdFromContext(r.Context()); !authenticated {
		if err := decodeJSON(r, &request); err != nil {
			writeInvalid(w, "INVALID_FORMAT", "format is invalid")
			return
		}
	}
	h.registerCustomerQueue(w, r, businessID, request)
}

func (h *QueueHandlerImpl) registerCustomerQueue(w http.ResponseWriter, r *http.Request, businessID uuid.UUID, request CustomerJoinRequest) {
	var userID *uuid.UUID
	if userIDString, authenticated := middleware.OptionalUserIdFromContext(r.Context()); authenticated {
		parsed, ok := parseUUIDValue(w, userIDString, "INVALID_USER_ID", "invalid user id")
		if !ok {
			return
		}
		userID = &parsed
	}

	// Check signing configuration before creating a ticket.
	if _, err := middleware.CreateQueueToken("", uuid.Nil.String()); err != nil {
		httpadapter.WriteError(w, apperror.Wrap(apperror.KindInternal, "CREATE_QUEUE_TOKEN_ERROR", "failed to create queue token", err))
		return
	}
	loadedUser, _ := middleware.LoadedUserFromContext(r.Context())
	result, err := h.s.RegisterCustomerQueue(r.Context(), app.RegisterCustomerQueueInput{
		BusinessID:        businessID,
		UserID:            userID,
		AuthenticatedUser: loadedUser,
		Username:          request.Username,
		PhoneNumber:       request.PhoneNumber,
	})
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	// Create and set the queue token.
	maxAge := int(config.GuestExpirationTime.Seconds())
	queueToken, err := middleware.CreateQueueToken(result.Username, result.Queue.ID.String())
	if err != nil {
		httpadapter.WriteError(w, apperror.Wrap(apperror.KindInternal, "CREATE_QUEUE_TOKEN_ERROR", "failed to create queue token", err))
		return
	}
	setTicketCookie(w, r, "queueToken", queueToken, result.Queue.ID.String(), maxAge)

	// Create and set the guest token.
	if result.GuestID != nil {
		guestToken, err := middleware.CreateGuestToken(result.GuestID.String(), businessID.String(), result.PhoneNumber)
		if err != nil {
			httpadapter.WriteError(w, apperror.Wrap(apperror.KindInternal, "CREATE_GUEST_TOKEN_ERROR", "failed to create guest token", err))
			return
		}
		setTicketCookie(w, r, "guestToken", guestToken, result.Queue.ID.String(), maxAge)
	}

	httpadapter.WriteJSON(w, http.StatusCreated, NewQueueResponse(result.Queue))
}

func setTicketCookie(w http.ResponseWriter, r *http.Request, name, value, queueID string, maxAge int) {
	// Per-ticket names preserve guest tickets even behind the /api reverse proxy.
	// Legacy root names retain QR compatibility for the newest ticket.
	secure := config.IsCookieSecure() || r.TLS != nil
	for _, cookieName := range []string{name + "_" + queueID, name} {
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: value, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
	}
}

func (h *QueueHandlerImpl) RegisterQueueByQr(w http.ResponseWriter, r *http.Request) {
	h.JoinBusinessQueue(w, r)
}

// Legacy POST /queues/ uses the same contract; userId/name/priority are rejected.
func (h *QueueHandlerImpl) RegisterQueue(w http.ResponseWriter, r *http.Request) {
	var request CreateQueueRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	businessID, ok := parseUUIDValue(w, request.BusinessID, "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}
	h.registerCustomerQueue(w, r, businessID, CustomerJoinRequest{Username: request.Username, PhoneNumber: request.PhoneNumber})
}

func (h *QueueHandlerImpl) GetQueueNameFromSnapShot(w http.ResponseWriter, r *http.Request) {
}

func (h *QueueHandlerImpl) GetQueue(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	if preloaded, loaded := middleware.LoadedQueueFromContext(r.Context()); loaded && preloaded.ID == queueID {
		httpadapter.WriteJSON(w, http.StatusOK, NewQueueResponse(preloaded))
		return
	}

	queue, err := h.s.GetQueue(r.Context(), queueID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, NewQueueResponse(queue))
}

func (h *QueueHandlerImpl) GetCustomerQueueStatus(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}
	preloaded, _ := middleware.LoadedQueueFromContext(r.Context())
	status, err := h.s.GetCustomerQueueStatusWithQueue(r.Context(), queueID, preloaded)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, NewCustomerQueueStatusResponse(status))
}

func (h *QueueHandlerImpl) GetQueueState(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	if preloaded, loaded := middleware.LoadedQueueFromContext(r.Context()); loaded && preloaded.ID == queueID {
		httpadapter.WriteJSON(w, http.StatusOK, QueueStateResponse{State: string(preloaded.State)})
		return
	}

	state, err := h.s.GetQueueState(r.Context(), queueID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, QueueStateResponse{State: string(state)})
}

func (h *QueueHandlerImpl) GetAllQueueByBusiness(w http.ResponseWriter, r *http.Request) {
	businessID, ok := parseUUIDParam(w, r, "businessID", "INVALID_BUSINESS_ID", "invalid business id")
	if !ok {
		return
	}

	queues, err := h.s.GetAllQueueByBusiness(r.Context(), businessID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, NewQueueResponses(queues))
}

func (h *QueueHandlerImpl) GetMyQueues(w http.ResponseWriter, r *http.Request) {
	identity, authenticated := middleware.UserIdFromContext(r.Context())
	userID, err := uuid.Parse(identity)
	if !authenticated || err != nil {
		httpadapter.WriteError(w, apperror.New(apperror.KindUnauthorized, "AUTHENTICATION_REQUIRED", "authentication required"))
		return
	}
	queues, err := h.s.GetActiveQueuesByUser(r.Context(), userID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, NewQueueResponses(queues))
}

func (h *QueueHandlerImpl) GetGuestQueues(w http.ResponseWriter, r *http.Request) {
	queues, err := h.resolveGuestQueuesFromRequest(r)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, NewQueueResponses(queues))
}

func (h *QueueHandlerImpl) GetGuestActiveQueue(w http.ResponseWriter, r *http.Request) {
	queues, err := h.resolveGuestQueuesFromRequest(r)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	if len(queues) == 0 {
		httpadapter.WriteError(w, apperror.New(apperror.KindNotFound, "ACTIVE_QUEUE_NOT_FOUND", "no active queue found for guest"))
		return
	}
	status, err := h.s.GetCustomerQueueStatus(r.Context(), queues[0].ID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, NewCustomerQueueStatusResponse(status))
}

func (h *QueueHandlerImpl) UpdateQueue(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	var request UpdateQueueRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}
	if request.Name == nil && request.State == nil && request.Priority == nil {
		writeInvalid(w, "NO_QUEUE_FIELDS", "no queue fields provided")
		return
	}
	// Ownership grants customer ticket access, never scheduler/staff authority.
	if request.Name != nil || request.Priority != nil || request.State == nil || *request.State != string(domain.QueueStateCancelled) {
		customerMutationDenied(w)
		return
	}
	if !h.cancelCustomerQueue(w, r, queueID) {
		return
	}
	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "queue updated"})
}

func (h *QueueHandlerImpl) UpdateState(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}

	var request UpdateQueueStateRequest
	if err := decodeJSON(r, &request); err != nil {
		writeInvalid(w, "INVALID_FORMAT", "format is invalid")
		return
	}

	state, ok := parseQueueState(w, request.State)
	if !ok {
		return
	}
	if state != domain.QueueStateCancelled {
		customerMutationDenied(w)
		return
	}
	if !h.cancelCustomerQueue(w, r, queueID) {
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "queue state updated"})
}

func (h *QueueHandlerImpl) MarkAsDone(w http.ResponseWriter, r *http.Request) {
	customerMutationDenied(w)
}

func customerMutationDenied(w http.ResponseWriter) {
	httpadapter.WriteError(w, apperror.New(apperror.KindForbidden, "QUEUE_CUSTOMER_MUTATION_DENIED", "customers may only cancel a waiting ticket; staff workflow controls service states"))
}

func (h *QueueHandlerImpl) cancelCustomerQueue(w http.ResponseWriter, r *http.Request, queueID uuid.UUID) bool {
	if err := h.s.CancelWaitingQueue(r.Context(), queueID); err != nil {
		httpadapter.WriteError(w, err)
		return false
	}
	return true
}

func (h *QueueHandlerImpl) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	queueID, ok := parseUUIDParam(w, r, "queueID", "INVALID_QUEUE_ID", "invalid queue id")
	if !ok {
		return
	}
	queue, err := h.s.GetQueue(r.Context(), queueID)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	if queue.State != domain.QueueStateWaiting && !queue.IsTerminal() {
		httpadapter.WriteError(w, apperror.New(apperror.KindConflict, "QUEUE_NOT_TERMINAL", "cancel a waiting queue before deleting the ticket"))
		return
	}
	if queue.State == domain.QueueStateWaiting {
		// Win the waiting-state transition before hard deletion, so a concurrent
		// Call Next cannot assign a ticket that is about to disappear.
		if err := h.s.CancelWaitingQueue(r.Context(), queueID); err != nil {
			httpadapter.WriteError(w, err)
			return
		}
	}

	if err := h.s.DeleteQueue(r.Context(), queueID); err != nil {
		httpadapter.WriteError(w, err)
		return
	}

	httpadapter.WriteJSON(w, http.StatusOK, map[string]string{"message": "queue deleted"})
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, name string, code string, message string) (uuid.UUID, bool) {
	return parseUUIDValue(w, chi.URLParam(r, name), code, message)
}

func parseUUIDValue(w http.ResponseWriter, value string, code string, message string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		writeInvalid(w, code, message)
		return uuid.Nil, false
	}
	return id, true
}

func parseQueueState(w http.ResponseWriter, value string) (domain.QueueState, bool) {
	state := domain.QueueState(strings.TrimSpace(value))
	switch state {
	case domain.QueueStateWaiting,
		domain.QueueStateCalled,
		domain.QueueStateProcessing,
		domain.QueueStateCancelled,
		domain.QueueStateSkipped,
		domain.QueueStateCompleted:
		return state, true
	default:
		writeInvalid(w, "INVALID_QUEUE_STATE", "invalid queue state")
		return "", false
	}
}

func writeInvalid(w http.ResponseWriter, code string, message string) {
	httpadapter.WriteError(w, apperror.New(
		apperror.KindInvalid,
		code,
		message,
	))
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
