package app_test

import (
	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	"QueueLite/internal/config"
	queueapp "QueueLite/internal/queue/app"
	"QueueLite/internal/queue/domain"
	"QueueLite/internal/testutil"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"sync"
	"testing"
)

func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != code {
		t.Fatalf("wanted %s got %v", code, err)
	}
}
func guestInput(f *testutil.QueueFixture) queueapp.RegisterCustomerQueueInput {
	return queueapp.RegisterCustomerQueueInput{BusinessID: f.Business.ID, Username: "  Guest  ", PhoneNumber: "+62 812-3456-7890"}
}

func TestCustomerValidation(t *testing.T) {
	for _, code := range []string{"USERNAME_REQUIRED", "PHONE_NUMBER_REQUIRED", "INVALID_PHONE_NUMBER", "BUSINESS_NOT_FOUND", "BUSINESS_UNAVAILABLE", "BUSINESS_QUEUE_FULL", "GET_BUSINESS_ERROR"} {
		t.Run(code, func(t *testing.T) {
			f := testutil.NewQueueFixture(t)
			input := guestInput(f)
			switch code {
			case "USERNAME_REQUIRED":
				input.Username = " "
			case "PHONE_NUMBER_REQUIRED":
				input.PhoneNumber = ""
			case "INVALID_PHONE_NUMBER":
				input.PhoneNumber = "123"
			case "BUSINESS_NOT_FOUND":
				input.BusinessID = uuid.New()
			case "BUSINESS_UNAVAILABLE":
				f.Business.Operational = false
			case "BUSINESS_QUEUE_FULL":
				f.Capacity = 0
			case "GET_BUSINESS_ERROR":
				f.BusinessError = errors.New("database private detail")
			}
			_, err := f.Service.RegisterCustomerQueue(context.Background(), input)
			assertCode(t, err, code)
			if f.Server.Exists(config.StreamName) {
				t.Fatal("failed registration enqueued")
			}
		})
	}
}

func TestCustomerIdentityUUIDAndDuplicates(t *testing.T) {
	ctx := context.Background()
	f := testutil.NewQueueFixture(t)
	input := guestInput(f)
	input.UserID = &f.User.ID
	input.Username = "spoofed"
	input.PhoneNumber = "123"
	result, err := f.Service.RegisterCustomerQueue(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	q := result.Queue
	if result.GuestID != nil || *q.UserID != f.User.ID || result.Username != f.User.Username || result.PhoneNumber != "6281234567890" || q.Priority {
		t.Fatalf("server identity/defaults: %#v", result)
	}
	_, err = f.Service.RegisterCustomerQueue(ctx, input)
	assertCode(t, err, "ACTIVE_QUEUE_EXISTS")
	if f.Capacity != 99 {
		t.Fatalf("duplicate consumed capacity: %d", f.Capacity)
	}
	entries, err := f.Redis.XRange(ctx, config.StreamName, "-", "+").Result()
	if err != nil || len(entries) != 1 {
		t.Fatalf("stream: %v %v", entries, err)
	}
	parsed, err := queueapp.QueueFromStreamValues(entries[0].Values)
	if err != nil || parsed.ID != q.ID {
		t.Fatalf("stream UUID changed: %v %v", parsed, err)
	}
	name, err := cache.GetQueueName(ctx, f.Redis, q.ID.String())
	if err != nil || name != q.Name {
		t.Fatal("cache UUID/name mismatch")
	}
	waiting, err := cache.GetQueue(ctx, cache.WaitingQueueKey(q.BusinessID.String()), f.Redis)
	if err != nil || len(waiting) != 1 || waiting[0].Member != q.ID.String() {
		t.Fatal("waiting UUID mismatch")
	}
	read, err := f.Service.GetQueue(ctx, q.ID)
	if err != nil || read.ID != q.ID {
		t.Fatal("immediate read failed")
	}
	customer, err := cache.GetCustomerQueue(ctx, f.Redis, q.ID.String())
	if err != nil || customer.Queue.ID != q.ID {
		t.Fatal("ownership UUID mismatch")
	}
	status, err := cache.EnqueueCustomerQueue(ctx, f.Redis, *customer)
	if err != nil || status != "OK" || f.Redis.XLen(ctx, config.StreamName).Val() != 1 {
		t.Fatal("enqueue retry not idempotent")
	}
}

func TestGuestPhoneLifecycle(t *testing.T) {
	for _, state := range []domain.QueueState{domain.QueueStateWaiting, domain.QueueStateCalled, domain.QueueStateProcessing, domain.QueueStateCompleted, domain.QueueStateCancelled, domain.QueueStateSkipped, "deleted"} {
		t.Run(string(state), func(t *testing.T) {
			ctx := context.Background()
			f := testutil.NewQueueFixture(t)
			input := guestInput(f)
			result, err := f.Service.RegisterQueueByQR(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			if result.GuestID == nil || result.Username != "Guest" {
				t.Fatal("guest identity not accepted")
			}
			f.Persist(*result.Queue)
			if state != domain.QueueStateWaiting {
				for _, key := range []string{cache.PriorityWaitingQueueKey(f.Business.ID.String()), cache.NormalWaitingQueueKey(f.Business.ID.String()), cache.LegacyWaitingQueueKey(f.Business.ID.String())} {
					if err := cache.AddQueue(ctx, f.Redis, key, *result.Queue); err != nil {
						t.Fatal(err)
					}
				}
			}
			if state == "deleted" {
				err = f.Service.DeleteQueue(ctx, result.Queue.ID)
			} else {
				err = f.Service.UpdateState(ctx, result.Queue.ID, state)
			}
			if err != nil {
				t.Fatal(err)
			}
			if state != domain.QueueStateWaiting {
				for _, key := range []string{cache.PriorityWaitingQueueKey(f.Business.ID.String()), cache.NormalWaitingQueueKey(f.Business.ID.String()), cache.LegacyWaitingQueueKey(f.Business.ID.String())} {
					if err := f.Redis.ZScore(ctx, key, result.Queue.ID.String()).Err(); err != redis.Nil {
						t.Fatalf("queue remains in %s after %s: %v", key, state, err)
					}
				}
			}
			_, err = f.Service.RegisterCustomerQueue(ctx, input)
			terminal := state == "deleted" || (domain.Queue{State: state}).IsTerminal()
			if terminal {
				if err != nil {
					t.Fatal(err)
				}
				if err := cache.ReleaseCustomerQueue(ctx, f.Redis, *result.Queue); err != nil {
					t.Fatal(err)
				}
				if !f.Server.Exists(cache.GuestPhoneKey(f.Business.ID.String(), result.PhoneNumber)) {
					t.Fatal("old cleanup erased new reservation")
				}
			} else {
				assertCode(t, err, "PHONE_ALREADY_REGISTERED")
			}
		})
	}
}

func TestWaitingSnapshotIncludesSplitAndLegacyQueues(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	ctx := context.Background()
	joined, err := f.Service.RegisterCustomerQueue(ctx, guestInput(f))
	if err != nil {
		t.Fatal(err)
	}
	if joined.Queue.Priority {
		t.Fatal("public customer join became priority")
	}
	want := map[string]string{joined.Queue.ID.String(): joined.Queue.Name}
	for i, key := range []string{cache.PriorityWaitingQueueKey(f.Business.ID.String()), cache.NormalWaitingQueueKey(f.Business.ID.String()), cache.LegacyWaitingQueueKey(f.Business.ID.String())} {
		q := *joined.Queue
		q.ID, q.Name = uuid.New(), fmt.Sprintf("A%03d", i+2)
		if err := cache.AddQueue(ctx, f.Redis, key, q); err != nil {
			t.Fatal(err)
		}
		want[q.ID.String()] = q.Name
	}
	snapshot, err := f.Service.GetWaitingQueueSnapshot(ctx, f.Business.ID)
	if err != nil || len(snapshot) != len(want) {
		t.Fatalf("waiting snapshot: %#v %v", snapshot, err)
	}
	for _, item := range snapshot {
		if name, exists := want[item.QueueID]; !exists || name != item.QueueName {
			t.Fatalf("snapshot lost stable ticket identity/name: %#v", item)
		}
		delete(want, item.QueueID)
	}
	if len(want) != 0 {
		t.Fatalf("missing waiting customers: %#v", want)
	}
}

func TestConcurrentCustomerQuotaAndPhone(t *testing.T) {
	for _, samePhone := range []bool{false, true} {
		f := testutil.NewQueueFixture(t)
		f.Capacity = 1
		if samePhone {
			f.Capacity = 100
		}
		var wg sync.WaitGroup
		successes := make(chan bool, 16)
		for i := 0; i < 16; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				input := guestInput(f)
				if !samePhone {
					input.PhoneNumber = fmt.Sprintf("08123456%04d", i)
				}
				_, err := f.Service.RegisterCustomerQueue(context.Background(), input)
				successes <- err == nil
			}(i)
		}
		wg.Wait()
		close(successes)
		count := 0
		for ok := range successes {
			if ok {
				count++
			}
		}
		remaining := 0
		if samePhone {
			remaining = 99
		}
		if count != 1 || f.Capacity != remaining || f.Redis.XLen(context.Background(), config.StreamName).Val() != 1 {
			t.Fatalf("quota/duplicate race: %d successes capacity %d", count, f.Capacity)
		}
	}
}

func TestPendingMutationDoesNotReleaseOrDelete(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	ctx := context.Background()
	result, err := f.Service.RegisterCustomerQueue(ctx, guestInput(f))
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func() error{
		func() error { return f.Service.DeleteQueue(ctx, result.Queue.ID) },
		func() error { return f.Service.MarkAsCancelled(ctx, result.Queue.ID) },
		func() error { return f.Service.MarkAsDone(ctx, result.Queue.ID) },
	} {
		assertCode(t, mutate(), "QUEUE_PERSISTENCE_PENDING")
	}
	if !f.Server.Exists(cache.GuestPhoneKey(f.Business.ID.String(), result.PhoneNumber)) {
		t.Fatal("pending mutation released reservation")
	}
}

func TestWrongRedisKeyTypeDoesNotEnqueueOrConsumeQuota(t *testing.T) {
	f := testutil.NewQueueFixture(t)
	ctx := context.Background()
	f.Redis.Set(ctx, cache.WaitingQueueKey(f.Business.ID.String()), "wrong-type", 0)
	_, err := f.Service.RegisterCustomerQueue(ctx, guestInput(f))
	assertCode(t, err, "REGISTER_QUEUE_ERROR")
	if f.Capacity != 100 || f.Redis.XLen(ctx, config.StreamName).Val() != 0 {
		t.Fatal("partial Redis registration")
	}
}
