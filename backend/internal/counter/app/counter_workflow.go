package app

import (
	"context"
	"errors"
	"log"
	"time"

	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	telebotapp "QueueLite/internal/bot/app"
	"QueueLite/internal/counter/domain"
	queueapp "QueueLite/internal/queue/app"
	queuedomain "QueueLite/internal/queue/domain"

	"github.com/google/uuid"
)

func (s *CounterService) UpdateCounterCustomer(ctx context.Context, counterID uuid.UUID, queueID uuid.UUID) error {
	counter, err := s.GetCounter(ctx, counterID)
	if err != nil {
		return err
	}

	queue, err := s.getQueue(ctx, queueID)
	if err != nil {
		return err
	}
	if queue.BusinessID != counter.BusinessID {
		return apperror.New(apperror.KindInvalid, "COUNTER_QUEUE_BUSINESS_MISMATCH", "counter and queue belong to different businesses")
	}

	wasWaiting := queue.State == "" || queue.State == queuedomain.QueueStateWaiting
	if wasWaiting {
		queue.State = queuedomain.QueueStateCalled
		queue.CalledByCounterID = &counterID
		queue.CalledAt = nowPtr()
	}

	if err := s.runInTx(ctx, func(counterRepo CounterRepo, queueRepo CounterQueueRepo) error {
		if err := queueRepo.UpdateQueue(ctx, queue); err != nil {
			return apperror.Wrap(apperror.KindInternal, "UPDATE_COUNTER_CUSTOMER_QUEUE_ERROR", "failed to assign queue to counter", err)
		}
		if err := counterRepo.UpdateCounterCustomer(ctx, counterID, &queueID); err != nil {
			return wrapCounterError(err, "UPDATE_COUNTER_CUSTOMER_ERROR", "failed to update counter customer")
		}
		return nil
	}); err != nil {
		return err
	}

	if wasWaiting {
		s.removeWaitingQueue(ctx, queue.BusinessID, queue.ID)
		s.publishQueueUpdate(ctx, queue.BusinessID)
	}
	return nil
}

func (s *CounterService) RemoveQueueFromCounter(ctx context.Context, counterID uuid.UUID, queueID uuid.UUID) error {
	counter, err := s.GetCounter(ctx, counterID)
	if err != nil {
		return err
	}

	queue, err := s.getQueue(ctx, queueID)
	if err != nil {
		return err
	}
	if queue.CalledByCounterID == nil || *queue.CalledByCounterID != counterID {
		return apperror.New(apperror.KindInvalid, "QUEUE_COUNTER_MISMATCH", "queue was not called by this counter")
	}

	queue.CalledByCounterID = nil
	queue.State = queuedomain.QueueStateCompleted
	queue.DoneAt = nowPtr()
	clearCounter := counter.CurrentQueueID != nil && *counter.CurrentQueueID == queueID

	if err := s.runInTx(ctx, func(counterRepo CounterRepo, queueRepo CounterQueueRepo) error {
		if err := queueRepo.UpdateQueue(ctx, queue); err != nil {
			return apperror.Wrap(apperror.KindInternal, "REMOVE_QUEUE_COUNTER_ERROR", "failed to remove queue from counter", err)
		}
		if clearCounter {
			if err := counterRepo.UpdateCounterCustomer(ctx, counterID, nil); err != nil {
				return wrapCounterError(err, "UPDATE_COUNTER_CUSTOMER_ERROR", "failed to update counter customer")
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if err := cache.SyncCustomerQueue(ctx, s.rdt, *queue); err != nil {
		return apperror.Wrap(apperror.KindInternal, "RELEASE_QUEUE_PHONE_ERROR", "failed to release queue reservation", err)
	}

	s.removeWaitingQueue(ctx, queue.BusinessID, queue.ID)
	s.publishQueueUpdate(ctx, queue.BusinessID)
	return nil
}

func (s *CounterService) CallNextQueue(ctx context.Context, counterID uuid.UUID, businessID uuid.UUID) (*queuedomain.Queue, error) {
	counter, err := s.GetCounter(ctx, counterID)
	if err != nil {
		return nil, err
	}
	if counter.BusinessID != businessID {
		return nil, apperror.New(apperror.KindInvalid, "COUNTER_BUSINESS_MISMATCH", "counter does not belong to business")
	}
	if err := s.completeCurrentCounterQueue(ctx, counter, businessID); err != nil {
		return nil, err
	}

	next, queueItem, err := s.popNextWaitingQueue(ctx, businessID)
	if err != nil || next == nil {
		return nil, err
	}

	if err := s.assignCalledQueueToCounter(ctx, counterID, next, *queueItem); err != nil {
		return nil, err
	}

	s.enqueueCallNotifications(ctx, next)
	s.publishQueueUpdate(ctx, next.BusinessID)
	s.scheduleCalledQueueTimeout(next.ID, counterID)
	return next, nil
}

func (s *CounterService) completeCurrentCounterQueue(ctx context.Context, counter *domain.Counter, businessID uuid.UUID) error {
	if counter.CurrentQueueID == nil {
		return nil
	}

	current, getErr := s.queueRepo.GetQueue(ctx, *counter.CurrentQueueID)
	var completedCurrent *queuedomain.Queue
	if err := s.runInTx(ctx, func(counterRepo CounterRepo, queueRepo CounterQueueRepo) error {
		if getErr == nil && current != nil && !current.IsTerminal() {
			current.State = queuedomain.QueueStateCompleted
			current.DoneAt = nowPtr()
			if err := queueRepo.UpdateQueue(ctx, current); err != nil {
				return err
			}
			completedCurrent = current
		}
		return counterRepo.UpdateCounterCustomer(ctx, counter.ID, nil)
	}); err != nil {
		return err
	}

	if completedCurrent != nil {
		if err := cache.SyncCustomerQueue(ctx, s.rdt, *completedCurrent); err != nil {
			return err
		}
		if err := cache.RemoveWaitingQueue(ctx, s.rdt, businessID.String(), completedCurrent.ID.String()); err != nil {
			return err
		}
	}
	return nil
}

func (s *CounterService) popNextWaitingQueue(ctx context.Context, businessID uuid.UUID) (*queuedomain.Queue, *cache.WaitingQueueItem, error) {
	queueItem, err := cache.PopNextFairWaitingQueueWithScore(ctx, s.rdt, businessID.String())
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.KindInternal, "POP_TOP_QUEUE_ERROR", "failed to pop top waiting queue", err)
	}
	if queueItem == nil {
		return nil, nil, nil
	}

	queueIDString, ok := queueItem.Member.(string)
	if !ok {
		return nil, nil, s.restoreWaitingQueueAfterFailure(ctx, *queueItem, apperror.New(apperror.KindInternal, "INVALID_QUEUE_CACHE_ID", "invalid queue id in waiting cache"))
	}
	queueID, err := uuid.Parse(queueIDString)
	if err != nil {
		return nil, nil, s.restoreWaitingQueueAfterFailure(ctx, *queueItem, apperror.Wrap(apperror.KindInternal, "INVALID_QUEUE_CACHE_ID", "invalid queue id in waiting cache", err))
	}

	next, err := s.getQueueWithRetry(ctx, queueID)
	if err != nil {
		return nil, nil, s.restoreWaitingQueueAfterFailure(ctx, *queueItem, err)
	}
	if next.BusinessID != businessID || next.State != queuedomain.QueueStateWaiting {
		// A stale/mis-scoped entry must never resurrect or disclose this queue.
		return nil, nil, apperror.New(apperror.KindConflict, "QUEUE_STATE_CHANGED", "waiting queue state changed; refresh before calling again")
	}
	return next, queueItem, nil
}

func (s *CounterService) assignCalledQueueToCounter(
	ctx context.Context,
	counterID uuid.UUID,
	next *queuedomain.Queue,
	queueItem cache.WaitingQueueItem,
) error {
	previous := *next
	next.State = queuedomain.QueueStateCalled
	next.CalledByCounterID = &counterID
	next.CalledAt = nowPtr()

	var queueUpdated bool
	var counterAssignErr error
	txErr := s.runInTx(ctx, func(counterRepo CounterRepo, queueRepo CounterQueueRepo) error {
		if err := queueRepo.UpdateQueueIfState(ctx, next, queuedomain.QueueStateWaiting); err != nil {
			return err
		}
		queueUpdated = true
		if err := counterRepo.UpdateCounterCustomer(ctx, counterID, &next.ID); err != nil {
			counterAssignErr = err
			return err
		}
		return nil
	})
	if txErr == nil {
		return nil
	}

	if !queueUpdated {
		var appErr *apperror.Error
		if errors.As(txErr, &appErr) && appErr.Code == "QUEUE_STATE_CHANGED" {
			return txErr
		}
		return s.restoreWaitingQueueAfterFailure(ctx, queueItem, apperror.Wrap(apperror.KindInternal, "CALL_NEXT_QUEUE_ERROR", "failed to call next queue", txErr))
	}

	recoveryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	// When running without a DB transaction manager (e.g. in-memory repos), roll back explicitly.
	if s.txManager == nil && counterAssignErr != nil {
		if rollbackErr := s.queueRepo.UpdateQueueIfState(recoveryCtx, &previous, queuedomain.QueueStateCalled); rollbackErr != nil {
			return errors.Join(txErr, rollbackErr)
		}
	}
	return s.restoreWaitingQueueAfterFailure(recoveryCtx, queueItem, txErr)
}

func (s *CounterService) enqueueCallNotifications(ctx context.Context, called *queuedomain.Queue) {
	if s.notifier == nil || called == nil {
		return
	}

	// Notify the customer who was just called.
	if called.UserID != nil && *called.UserID != uuid.Nil {
		if err := s.notifier.EnqueueUserNotification(ctx, *called.UserID, telebotapp.CurrentQueueNotificationMessage); err != nil {
			log.Printf("[COUNTER WARN] failed to enqueue called notification for queue=%s: %v", called.ID, err)
		}
	}

	s.notifyNextInLine(ctx, called.BusinessID)
}

func (s *CounterService) notifyNextInLine(ctx context.Context, businessID uuid.UUID) {
	if s.rdt == nil {
		return
	}
	snapshot, err := cache.GetWaitingQueueSnapshot(ctx, s.rdt, businessID.String())
	if err != nil {
		log.Printf("[COUNTER WARN] failed to load waiting snapshot for next notification: %v", err)
		return
	}
	if len(snapshot) == 0 {
		return
	}
	upcomingID, err := uuid.Parse(snapshot[0].QueueID)
	if err != nil {
		log.Printf("[COUNTER WARN] invalid next queue id %q in waiting snapshot: %v", snapshot[0].QueueID, err)
		return
	}
	upcoming, err := s.queueRepo.GetQueue(ctx, upcomingID)
	if err != nil || upcoming == nil || upcoming.UserID == nil || *upcoming.UserID == uuid.Nil {
		return
	}
	if err := s.notifier.EnqueueUserNotification(ctx, *upcoming.UserID, telebotapp.NextQueueNotificationMessage); err != nil {
		log.Printf("[COUNTER WARN] failed to enqueue next-in-line notification for queue=%s: %v", upcoming.ID, err)
	}
}

func (s *CounterService) restoreWaitingQueueAfterFailure(ctx context.Context, item cache.WaitingQueueItem, cause error) error {
	// The original request may have been cancelled while waiting for persistence.
	recoveryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := cache.RestoreWaitingQueueItem(recoveryCtx, s.rdt, item); err != nil {
		return apperror.Wrap(apperror.KindInternal, "RESTORE_WAITING_QUEUE_ERROR", "failed to restore waiting queue", errors.Join(cause, err))
	}
	return cause
}

func (s *CounterService) ProcessCalledQueue(ctx context.Context, counterID uuid.UUID, queueID uuid.UUID) (*queuedomain.Queue, error) {
	queue, err := s.getQueue(ctx, queueID)
	if err != nil {
		return nil, err
	}
	if queue.State != queuedomain.QueueStateCalled {
		return nil, apperror.New(apperror.KindInvalid, "QUEUE_NOT_CALLED", "queue is not called")
	}
	if queue.CalledByCounterID == nil || *queue.CalledByCounterID != counterID {
		return nil, apperror.New(apperror.KindInvalid, "QUEUE_COUNTER_MISMATCH", "queue was not called by this counter")
	}

	slotDecreased := false
	if s.subscriptionService != nil && queue.Priority && queue.UserID != nil {
		if err := s.subscriptionService.DecreaseUserSlot(ctx, *queue.UserID); err != nil {
			return nil, err
		}
		slotDecreased = true
	}

	queue.State = queuedomain.QueueStateProcessing
	queue.ProcessingAt = nowPtr()

	if err := s.runInTx(ctx, func(counterRepo CounterRepo, queueRepo CounterQueueRepo) error {
		if err := queueRepo.UpdateQueue(ctx, queue); err != nil {
			return err
		}
		return counterRepo.UpdateCounterCustomer(ctx, counterID, &queueID)
	}); err != nil {
		if slotDecreased && queue.UserID != nil {
			s.refundPrioritySlot(ctx, *queue.UserID, queue.ID)
		}
		return nil, err
	}

	s.removeWaitingQueue(ctx, queue.BusinessID, queue.ID)
	s.publishQueueUpdate(ctx, queue.BusinessID)
	return queue, nil
}

func (s *CounterService) refundPrioritySlot(ctx context.Context, userID uuid.UUID, queueID uuid.UUID) {
	if s.subscriptionService == nil {
		return
	}
	refundCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if refundErr := s.subscriptionService.AddUserSlot(refundCtx, userID, 1); refundErr != nil {
		log.Printf("[COUNTER ERROR] failed to refund priority slot for user=%s queue=%s: %v", userID, queueID, refundErr)
	}
}

func (s *CounterService) SkipQueue(ctx context.Context, counterID uuid.UUID, queueID uuid.UUID) (*queuedomain.Queue, error) {
	queue, err := s.getQueue(ctx, queueID)
	if err != nil {
		return nil, err
	}
	if queue.CalledByCounterID == nil || *queue.CalledByCounterID != counterID {
		return nil, apperror.New(apperror.KindInvalid, "QUEUE_COUNTER_MISMATCH", "queue was not called by this counter")
	}
	if queue.State != queuedomain.QueueStateCalled && queue.State != queuedomain.QueueStateProcessing {
		return nil, apperror.New(apperror.KindInvalid, "QUEUE_INVALID_STATE", "queue cannot be skipped")
	}

	counter, err := s.GetCounter(ctx, counterID)
	if err != nil {
		return nil, err
	}

	queue.State = queuedomain.QueueStateSkipped
	clearCounter := counter.CurrentQueueID != nil && *counter.CurrentQueueID == queueID

	if err := s.runInTx(ctx, func(counterRepo CounterRepo, queueRepo CounterQueueRepo) error {
		if err := queueRepo.UpdateQueue(ctx, queue); err != nil {
			return err
		}
		if clearCounter {
			return counterRepo.UpdateCounterCustomer(ctx, counterID, nil)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if err := cache.SyncCustomerQueue(ctx, s.rdt, *queue); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "RELEASE_QUEUE_PHONE_ERROR", "failed to release queue reservation", err)
	}
	s.removeWaitingQueue(ctx, queue.BusinessID, queue.ID)

	next, err := s.CallNextQueue(ctx, counterID, queue.BusinessID)
	if err != nil {
		return nil, err
	}
	if next == nil {
		s.publishQueueUpdate(ctx, queue.BusinessID)
	}
	return next, nil
}

func (s *CounterService) getQueueWithRetry(ctx context.Context, queueID uuid.UUID) (*queuedomain.Queue, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		queue, err := s.queueRepo.GetQueue(ctx, queueID)
		if err == nil {
			return queue, nil
		}
		if !errors.Is(err, queueapp.ErrQueueNotFound) {
			return nil, err
		}
		lastErr = err
		if attempt < 2 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(3 * time.Second):
			}
		}
	}
	return nil, apperror.Wrap(apperror.KindNotFound, "QUEUE_NOT_READY", "queue is not ready in database yet", lastErr)
}

func (s *CounterService) scheduleCalledQueueTimeout(queueID uuid.UUID, counterID uuid.UUID) {
	time.AfterFunc(5*time.Minute, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if s.queueRepo == nil || s.repo == nil {
			return
		}
		// Coordinate with HTTP counter mutations via the same per-counter Redis lock.
		release, locked, err := cache.AcquireCounterLock(ctx, s.rdt, counterID.String(), cache.DefaultCounterLockTTL)
		if err != nil || !locked {
			return
		}
		defer release()

		counter, err := s.repo.GetCounter(ctx, counterID)
		if err != nil || counter == nil || counter.CurrentQueueID == nil || *counter.CurrentQueueID != queueID {
			return
		}
		queue, err := s.queueRepo.GetQueue(ctx, queueID)
		if err != nil || queue == nil || queue.State != queuedomain.QueueStateCalled || queue.CalledByCounterID == nil || *queue.CalledByCounterID != counterID {
			return
		}
		_, _ = s.SkipQueue(ctx, counterID, queueID)
	})
}
