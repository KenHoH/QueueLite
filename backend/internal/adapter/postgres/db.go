package postgres

import (
	"QueueLite/internal/adapter/postgres/model"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewConnection(database_url string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(database_url), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	return db
}

func MigrateDatabase(db *gorm.DB) error {
	err := db.AutoMigrate(&model.Business{}, &model.Counter{}, &model.Queue{}, &model.User{}, &model.Subscription{}, &model.SubscriptionPlan{}, &model.UserBusinessRelation{})
	return err
}
