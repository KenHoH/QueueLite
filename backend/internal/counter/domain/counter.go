package domain

import "github.com/google/uuid"

type Counter struct {
	ID                uuid.UUID
	BusinessID        uuid.UUID
	Name              string
	CurrentEmployeeID *uuid.UUID
	CurrentQueueID    *uuid.UUID
}
