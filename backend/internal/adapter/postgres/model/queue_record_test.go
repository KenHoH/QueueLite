package model

import (
	"github.com/google/uuid"
	"testing"
)

func TestQueueBeforeCreate(t *testing.T) {
	id := uuid.New()
	assigned := Queue{ID: id}
	if err := assigned.BeforeCreate(nil); err != nil || assigned.ID != id {
		t.Fatalf("assigned UUID changed: %v %v", assigned.ID, err)
	}
	empty := Queue{}
	if err := empty.BeforeCreate(nil); err != nil || empty.ID == uuid.Nil {
		t.Fatalf("nil UUID not generated: %v %v", empty.ID, err)
	}
}
