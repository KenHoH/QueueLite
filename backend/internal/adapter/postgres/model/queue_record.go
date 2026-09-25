package model

import (
	"time"

	"github.com/google/uuid"
)

type Queue struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	BusinessID uuid.UUID  `gorm:"type:uuid;not null;index:idx_queue_business_state_created,priority:1" json:"businessId"`
	UserID     *uuid.UUID `gorm:"type:uuid;index" json:"userId,omitempty"`

	CalledByCounterID *uuid.UUID `gorm:"type:uuid;index" json:"calledByCounterId,omitempty"`

	Name string `gorm:"size:100;not null" json:"name"`

	State QueueState `gorm:"type:varchar(20);not null;default:waiting;index:idx_queue_business_state_created,priority:2" json:"state"`

	Priority bool `gorm:"not null;default:false;index" json:"priority"`

	CreatedAt time.Time `gorm:"index:idx_queue_business_state_created,priority:3" json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Business        Business `gorm:"foreignKey:BusinessID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	CalledByCounter *Counter `gorm:"foreignKey:CalledByCounterID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"calledByCounter,omitempty"`
}
