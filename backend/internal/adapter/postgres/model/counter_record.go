package model

import (
	"time"

	"github.com/google/uuid"
)

type Counter struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BusinessID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_business_counter_name" json:"businessId"`

	Name string `gorm:"size:100;not null;uniqueIndex:idx_business_counter_name" json:"name"`

	// Nullable because a counter might not currently have an employee.
	CurrentEmployeeID *uuid.UUID `gorm:"type:uuid;index" json:"currentEmployeeId,omitempty"`

	// Nullable because a counter might not currently serve a queue.
	CurrentQueueID *uuid.UUID `gorm:"type:uuid;index" json:"currentQueueId,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Business        Business `gorm:"foreignKey:BusinessID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	CurrentEmployee *User    `gorm:"foreignKey:CurrentEmployeeID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"currentEmployee,omitempty"`
	CurrentQueue    *Queue   `gorm:"foreignKey:CurrentQueueID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"currentQueue,omitempty"`
}
