package model

import (
	"time"

	"github.com/google/uuid"
)

type Business struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name        string    `gorm:"column:business_name;size:150;not null" json:"name"`
	Location    string    `gorm:"column:business_location;size:255;not null" json:"location"`
	Description *string   `gorm:"column:business_description;type:text" json:"description,omitempty"`
	Operational bool      `gorm:"column:business_operational;not null;default:true" json:"operational"`
	Email       string    `gorm:"column:business_email;size:255;not null" json:"email"`
	PhoneNumber string    `gorm:"column:business_phone_number;size:30;not null" json:"phoneNumber"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	UserRelations []UserBusinessRelation `gorm:"foreignKey:BusinessID" json:"userRelations,omitempty"`
	Queues        []Queue                `gorm:"foreignKey:BusinessID" json:"queues,omitempty"`
	Counters      []Counter              `gorm:"foreignKey:BusinessID" json:"counters,omitempty"`

	// BusinessID has a unique index in Subscription,
	// making this a one-to-one relationship.
	Subscription *Subscription `gorm:"foreignKey:BusinessID" json:"subscription,omitempty"`
}
