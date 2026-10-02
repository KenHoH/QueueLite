// Package operations authorizes the staff HTTP boundary without changing queue scheduling.
package operations

import (
	httpadapter "QueueLite/internal/adapter/http"
	"QueueLite/internal/apperror"
	counterapp "QueueLite/internal/counter/app"
	counterdomain "QueueLite/internal/counter/domain"
	counterhttp "QueueLite/internal/counter/inbound/http"
	"QueueLite/internal/middleware"
	queueapp "QueueLite/internal/queue/app"
	queuedomain "QueueLite/internal/queue/domain"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"io"
	"net/http"
	"strings"
	"time"
)

type Member struct {
	UserID   uuid.UUID `json:"userId"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
}
type Businesses interface {
	GetUserBusinessRole(context.Context, uuid.UUID, uuid.UUID) (string, error)
	ListBusinessMembers(context.Context, uuid.UUID) ([]Member, error)
}
type Counters interface {
	GetCounter(context.Context, uuid.UUID) (*counterdomain.Counter, error)
	ListBusinessCounters(context.Context, uuid.UUID) ([]counterdomain.Counter, error)
}
type Queues interface {
	GetQueue(context.Context, uuid.UUID) (*queuedomain.Queue, error)
}
type Handler struct {
	businesses Businesses
	counters   Counters
	queues     Queues
	redis      *redis.Client
}

func New(b Businesses, c Counters, q Queues, r *redis.Client) *Handler { return &Handler{b, c, q, r} }

func denied() error {
	return apperror.New(apperror.KindForbidden, "BUSINESS_ACCESS_DENIED", "you cannot access this business resource")
}
func stale() error {
	return apperror.New(apperror.KindConflict, "COUNTER_STATE_CHANGED", "counter state changed; refresh before trying again")
}
func (h *Handler) role(r *http.Request, businessID uuid.UUID, management bool) (uuid.UUID, string, error) {
	value, ok := middleware.UserIdFromContext(r.Context())
	userID, err := uuid.Parse(value)
	if !ok || err != nil || userID == uuid.Nil {
		return uuid.Nil, "", apperror.New(apperror.KindUnauthorized, "UNAUTHORIZED", "sign in required")
	}
	role, err := h.businesses.GetUserBusinessRole(r.Context(), userID, businessID)
	if err != nil {
		return uuid.Nil, "", apperror.Wrap(apperror.KindInternal, "BUSINESS_ACCESS_ERROR", "failed to check membership", err)
	}
	if role != "owner" && role != "admin" && (management || role != "counter") {
		return uuid.Nil, "", denied()
	}
	return userID, role, nil
}
func param(r *http.Request, key string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperror.New(apperror.KindInvalid, "INVALID_RESOURCE_ID", "invalid resource id")
	}
	return id, nil
}
func (h *Handler) RequireBusiness(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := param(r, "businessID")
		if err == nil {
			_, _, err = h.role(r, id, false)
		}
		if err != nil {
			httpadapter.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (h *Handler) ListCounters(w http.ResponseWriter, r *http.Request) {
	id, err := param(r, "businessID")
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	user, role, err := h.role(r, id, false)
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	items, err := h.counters.ListBusinessCounters(r.Context(), id)
	if err != nil {
		httpadapter.WriteError(w, apperror.Wrap(apperror.KindInternal, "LIST_COUNTERS_ERROR", "failed to list counters", err))
		return
	}
	result := make([]counterhttp.CounterResponse, 0, len(items))
	for i := range items {
		if role != "counter" || (items[i].CurrentEmployeeID != nil && *items[i].CurrentEmployeeID == user) {
			result = append(result, counterhttp.NewCounterResponse(&items[i]))
		}
	}
	httpadapter.WriteJSON(w, http.StatusOK, result)
}
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	id, err := param(r, "businessID")
	if err == nil {
		_, _, err = h.role(r, id, true)
	}
	if err != nil {
		httpadapter.WriteError(w, err)
		return
	}
	items, err := h.businesses.ListBusinessMembers(r.Context(), id)
	if err != nil {
		httpadapter.WriteError(w, apperror.Wrap(apperror.KindInternal, "LIST_MEMBERS_ERROR", "failed to list members", err))
		return
	}
	if items == nil {
		items = []Member{}
	}
	httpadapter.WriteJSON(w, http.StatusOK, items)
}

// All counter reads/actions resolve their business on the server. Staff may only
// operate their assigned counters; owners/admins may operate all business counters.
func (h *Handler) RequireCounter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := param(r, "counterID")
		if err != nil {
			httpadapter.WriteError(w, err)
			return
		}
		counter, err := h.counters.GetCounter(r.Context(), id)
		if err != nil {
			httpadapter.WriteError(w, counterReadError(err))
			return
		}
		management := r.Method == http.MethodPut || (r.Method == http.MethodDelete && chi.URLParam(r, "queueID") == "")
		user, role, err := h.role(r, counter.BusinessID, management)
		if err == nil && role == "counter" && (counter.CurrentEmployeeID == nil || *counter.CurrentEmployeeID != user) {
			err = denied()
		}
		if err != nil {
			httpadapter.WriteError(w, err)
			return
		}
		// Serialize authorized HTTP mutations. Read state again under the lock.
		// Bound request duration below the lease; priority selection stays untouched.
		if r.Method != http.MethodGet && h.redis != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
			defer cancel()
			r = r.WithContext(ctx)
			key, token := "counter:operation:"+id.String(), uuid.NewString()
			locked, e := h.redis.SetNX(r.Context(), key, token, 30*time.Second).Result()
			if e != nil {
				httpadapter.WriteError(w, apperror.Wrap(apperror.KindInternal, "COUNTER_LOCK_ERROR", "counter operation unavailable", e))
				return
			}
			if !locked {
				httpadapter.WriteError(w, stale())
				return
			}
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				h.redis.Eval(ctx, `if redis.call('GET',KEYS[1]) == ARGV[1] then return redis.call('DEL',KEYS[1]) end return 0`, []string{key}, token)
			}()
			counter, err = h.counters.GetCounter(r.Context(), id)
			if err != nil {
				httpadapter.WriteError(w, counterReadError(err))
				return
			}
			user, role, err = h.role(r, counter.BusinessID, management)
			if err == nil && role == "counter" && (counter.CurrentEmployeeID == nil || *counter.CurrentEmployeeID != user) {
				err = denied()
			}
			if err != nil {
				httpadapter.WriteError(w, err)
				return
			}
		}
		if chi.URLParam(r, "businessID") != "" {
			businessID, e := param(r, "businessID")
			if e != nil {
				httpadapter.WriteError(w, e)
				return
			}
			if businessID != counter.BusinessID {
				httpadapter.WriteError(w, denied())
				return
			}
			if counter.CurrentQueueID != nil {
				httpadapter.WriteError(w, stale())
				return
			}
		}
		if chi.URLParam(r, "queueID") != "" {
			queueID, e := param(r, "queueID")
			if e != nil {
				httpadapter.WriteError(w, e)
				return
			}
			if counter.CurrentQueueID == nil || *counter.CurrentQueueID != queueID {
				httpadapter.WriteError(w, stale())
				return
			}
			queue, e := h.queues.GetQueue(r.Context(), queueID)
			if e != nil {
				if errors.Is(e, queueapp.ErrQueueNotFound) {
					httpadapter.WriteError(w, apperror.Wrap(apperror.KindNotFound, "QUEUE_NOT_READY", "queue details not available yet", e))
				} else {
					httpadapter.WriteError(w, apperror.Wrap(apperror.KindInternal, "GET_QUEUE_ERROR", "failed to read queue", e))
				}
				return
			}
			if queue.BusinessID != counter.BusinessID || queue.CalledByCounterID == nil || *queue.CalledByCounterID != id {
				httpadapter.WriteError(w, denied())
				return
			}
			if (strings.HasSuffix(r.URL.Path, "/process") && queue.State != queuedomain.QueueStateCalled) || (r.Method == http.MethodDelete && queue.State != queuedomain.QueueStateProcessing) || (strings.HasSuffix(r.URL.Path, "/skip") && queue.State != queuedomain.QueueStateCalled && queue.State != queuedomain.QueueStateProcessing) {
				httpadapter.WriteError(w, stale())
				return
			}
		}
		if management && r.Method == http.MethodDelete && counter.CurrentQueueID != nil {
			httpadapter.WriteError(w, stale())
			return
		}
		if r.Method == http.MethodPut {
			if err := h.validateBody(r, counter.BusinessID, false); err != nil {
				httpadapter.WriteError(w, err)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func counterReadError(err error) error {
	if errors.Is(err, counterapp.ErrCounterNotFound) {
		return apperror.Wrap(apperror.KindNotFound, "COUNTER_NOT_FOUND", "counter not found", err)
	}
	return apperror.Wrap(apperror.KindInternal, "GET_COUNTER_ERROR", "failed to read counter", err)
}

func (h *Handler) RequireCreate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h.validateBody(r, uuid.Nil, true); err != nil {
			httpadapter.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (h *Handler) validateBody(r *http.Request, businessID uuid.UUID, create bool) error {
	data, err := io.ReadAll(io.LimitReader(r.Body, 65537))
	if err != nil || len(data) > 65536 {
		return apperror.New(apperror.KindInvalid, "INVALID_FORMAT", "invalid request body")
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	var body struct {
		BusinessID string  `json:"businessId"`
		Name       *string `json:"name"`
		Employee   *string `json:"currentEmployeeId"`
		Queue      *string `json:"currentQueueId"`
	}
	if err = json.Unmarshal(data, &body); err != nil {
		return apperror.New(apperror.KindInvalid, "INVALID_FORMAT", "invalid request body")
	}
	if create {
		businessID, err = uuid.Parse(body.BusinessID)
		if err != nil || businessID == uuid.Nil {
			return apperror.New(apperror.KindInvalid, "INVALID_BUSINESS_ID", "invalid business id")
		}
		if _, _, err = h.role(r, businessID, true); err != nil {
			return err
		}
	}
	// Queue assignment belongs to atomic Call Next, never general counter editing.
	if body.Queue != nil {
		return apperror.New(apperror.KindInvalid, "COUNTER_QUEUE_ASSIGNMENT_UNSUPPORTED", "use counter workflow actions to change the customer")
	}
	if (create && body.Name == nil) || (body.Name != nil && strings.TrimSpace(*body.Name) == "") {
		return apperror.New(apperror.KindInvalid, "COUNTER_NAME_REQUIRED", "counter name required")
	}
	if body.Employee != nil && strings.TrimSpace(*body.Employee) != "" {
		employee, err := uuid.Parse(*body.Employee)
		if err != nil {
			return apperror.New(apperror.KindInvalid, "INVALID_EMPLOYEE_ID", "invalid employee id")
		}
		role, err := h.businesses.GetUserBusinessRole(r.Context(), employee, businessID)
		if err != nil {
			return apperror.Wrap(apperror.KindInternal, "BUSINESS_ACCESS_ERROR", "failed to check employee membership", err)
		}
		if role != "owner" && role != "admin" && role != "counter" {
			return denied()
		}
	}
	return nil
}
