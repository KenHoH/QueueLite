package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Queue struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	BusinessID uuid.UUID  `gorm:"type:uuid;not null;index:idx_queue_business_state_created,priority:1" json:"businessId"`
	UserID     *uuid.UUID `gorm:"type:uuid;index" json:"userId,omitempty"`

	CalledByCounterID *uuid.UUID `gorm:"type:uuid;index" json:"calledByCounterId,omitempty"`

	Name string `gorm:"size:100;not null" json:"name"`

	State QueueState `gorm:"type:varchar(20);not null;default:waiting;index:idx_queue_business_state_created,priority:2" json:"state"`

	Priority bool `gorm:"not null;default:false;index" json:"priority"`

	CalledAt     *time.Time `gorm:"type:timestamptz" json:"calledAt,omitempty"`
	ProcessingAt *time.Time `gorm:"type:timestamptz" json:"processingAt,omitempty"`
	DoneAt       *time.Time `gorm:"type:timestamptz" json:"doneAt,omitempty"`
	CancelledAt  *time.Time `gorm:"type:timestamptz" json:"cancelledAt,omitempty"`

	CreatedAt time.Time `gorm:"index:idx_queue_business_state_created,priority:3" json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Business        Business `gorm:"foreignKey:BusinessID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	CalledByCounter *Counter `gorm:"foreignKey:CalledByCounterID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"calledByCounter,omitempty"`
}

func (q *Queue) BeforeCreate(tx *gorm.DB) (err error) {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}
