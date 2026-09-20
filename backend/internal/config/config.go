package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func GetDatabaseUrl() string {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	name := os.Getenv("DATABASE_URL")
	return name
}
