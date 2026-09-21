package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Username    string    `gorm:"size:100;not null;uniqueIndex" json:"username"`
	PhoneNumber string    `gorm:"size:30;not null;uniqueIndex" json:"phoneNumber"`
	Password    *string   `gorm:"size:255;" json:"password"`
	Email       *string   `gorm:"size:255;uniqueIndex" json:"email,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// many to many relation
	BusinessRelations []UserBusinessRelation `gorm:"foreignKey:UserID" json:"businessRelations,omitempty"`

	Queues           []Queue       `gorm:"foreignKey:UserID" json:"queues,omitempty"`
	AssignedCounters []Counter     `gorm:"foreignKey:CurrentEmployeeID" json:"assignedCounters,omitempty"`
	Subscription     *Subscription `gorm:"foreignKey:BusinessID" json:"subscription,omitempty"` //optional that's why we use pointer
}
