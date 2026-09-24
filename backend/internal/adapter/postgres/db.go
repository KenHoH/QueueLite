package postgres

import (
	"QueueLite/internal/adapter/postgres/model"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewConnection(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	return db, nil
}

func MigrateDatabase(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.Business{},
		&model.Counter{},
		&model.Queue{},
		&model.User{},
		&model.BusinessPlan{},
		&model.UserPlan{},
		&model.Subscription{},
		&model.UserBusinessRelation{},
	); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	return nil
}
