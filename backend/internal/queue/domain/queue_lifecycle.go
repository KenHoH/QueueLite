package domain

import "time"

// StampLifecycle records the first observed lifecycle event, preserving earlier
// timestamps on retries and ordinary edits. Skipped has no cancellation event.
func (q *Queue) StampLifecycle(now time.Time) {
	var field **time.Time
	switch q.State {
	case QueueStateCalled:
		field = &q.CalledAt
	case QueueStateProcessing:
		field = &q.ProcessingAt
	case QueueStateCompleted:
		field = &q.DoneAt
	case QueueStateCancelled:
		field = &q.CancelledAt
	default:
		return
	}
	if *field == nil {
		*field = &now
	}
}
