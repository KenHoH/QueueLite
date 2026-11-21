package postgres

import (
	"QueueLite/internal/adapter/postgres/model"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PoolOptions struct {
	MaxOpenConns int
	MaxIdleConns int
}

func NewConnection(databaseURL string, opts ...PoolOptions) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		maxOpen, maxIdle := 25, 10
		if len(opts) > 0 {
			if opts[0].MaxOpenConns > 0 {
				maxOpen = opts[0].MaxOpenConns
			}
			if opts[0].MaxIdleConns > 0 {
				maxIdle = opts[0].MaxIdleConns
			}
		}
		sqlDB.SetMaxOpenConns(maxOpen)
		sqlDB.SetMaxIdleConns(maxIdle)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)
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
