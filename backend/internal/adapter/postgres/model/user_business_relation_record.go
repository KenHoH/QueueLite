package model

import (
	"time"

	"github.com/google/uuid"
)

type UserBusinessRelation struct {
	UserID     uuid.UUID    `gorm:"type:uuid;primaryKey;index" json:"userId"`
	BusinessID uuid.UUID    `gorm:"type:uuid;primaryKey;index" json:"businessId"`
	Role       BusinessRole `gorm:"type:varchar(20);not null" json:"role"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	User     User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Business Business `gorm:"foreignKey:BusinessID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

func (UserBusinessRelation) TableName() string {
	return "user_business_relations"
}
