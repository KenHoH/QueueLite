package model

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	BusinessID *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"businessId,omitempty"`
	UserID     *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"userId,omitempty"`

	BusinessPlanID *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"businessPlanId,omitempty"`
	UserPlanID     *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"userPlanId,omitempty"`

	Type SubscriptionType `gorm:"type:varchar(20);not null" json:"type"`

	StartDate time.Time  `gorm:"not null" json:"startDate"`
	EndDate   *time.Time `json:"endDate,omitempty"`

	Status SubscriptionStatus `gorm:"type:varchar(20);not null;default:active" json:"status"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Business     *Business     `gorm:"foreignKey:BusinessID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	User         *User         `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	BusinessPlan *BusinessPlan `gorm:"foreignKey:BusinessPlanID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"businessPlan,omitempty"`
	UserPlan     *UserPlan     `gorm:"foreignKey:UserPlanID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"userPlan,omitempty"`
}
