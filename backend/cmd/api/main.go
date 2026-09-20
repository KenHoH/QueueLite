package main

import (
	"QueueLite/internal/adapter/postgres"
	"QueueLite/internal/config"
	"fmt"
	"log"
)

func main() {
	url := config.GetDatabaseUrl()
	db := postgres.NewConnection(url)

	errMigration := postgres.MigrateDatabase(db)
	if errMigration != nil {
		log.Fatal("Database Migration failed")
	}
	fmt.Println("Testing")
}
