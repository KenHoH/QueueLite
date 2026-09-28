package postgres

import (
	"QueueLite/internal/adapter/postgres/model"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewConnection(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		// Queue and Counter reference each other, so inline FK creation can fail
		// during AutoMigrate when one table does not exist yet.
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	return db, nil
}

func MigrateDatabase(db *gorm.DB) error {
	if err := db.AutoMigrate(
		// Base tables first.
		&model.User{},
		&model.Business{},
		&model.BusinessPlan{},
		&model.UserPlan{},

		// Dependent tables after their main parent tables exist.
		&model.Subscription{},
		&model.Counter{},
		&model.Queue{},
		&model.UserBusinessRelation{},
	); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	return nil
}
