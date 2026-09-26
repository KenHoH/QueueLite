package app

import (
	"QueueLite/internal/queue/domain"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func QueueFromStreamValues(values map[string]any) (*domain.Queue, error) {
	queueID, err := requiredStreamString(values, "id")
	if err != nil {
		return nil, err
	}
	businessID, err := requiredStreamString(values, "business_id")
	if err != nil {
		return nil, err
	}
	name, err := requiredStreamString(values, "name")
	if err != nil {
		return nil, err
	}
	priorityRaw, err := requiredStreamString(values, "priority")
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(queueID)
	if err != nil {
		return nil, fmt.Errorf("parse queue id: %w", err)
	}
	businessUUID, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("parse business id: %w", err)
	}
	priority, err := strconv.ParseBool(priorityRaw)
	if err != nil {
		return nil, fmt.Errorf("parse priority: %w", err)
	}

	state := domain.QueueStateWaiting
	if stateRaw, ok := streamString(values, "state"); ok && stateRaw != "" {
		state = domain.QueueState(stateRaw)
	}

	var userID *uuid.UUID
	if userRaw, ok := streamString(values, "user_id"); ok && userRaw != "" {
		parsedUserID, err := uuid.Parse(userRaw)
		if err != nil {
			return nil, fmt.Errorf("parse user id: %w", err)
		}
		userID = &parsedUserID
	}

	createdAt := time.Now()
	if createdAtRaw, ok := streamString(values, "created_at"); ok && createdAtRaw != "" {
		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		createdAt = parsedCreatedAt
	}

	return &domain.Queue{
		ID:         id,
		BusinessID: businessUUID,
		UserID:     userID,
		Name:       name,
		State:      state,
		Priority:   priority,
		CreatedAt:  createdAt,
	}, nil
}

func requiredStreamString(values map[string]any, key string) (string, error) {
	value, ok := streamString(values, key)
	if !ok || value == "" {
		return "", fmt.Errorf("missing %s", key)
	}
	return value, nil
}

func streamString(values map[string]any, key string) (string, bool) {
	value, ok := values[key]
	if !ok || value == nil {
		return "", false
	}
	return fmt.Sprint(value), true
}
