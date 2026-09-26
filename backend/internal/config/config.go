package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	HTTPAddr    string
	SecretKey   string
	RedisAddr   string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	secretKey := os.Getenv("SECRET")
	if secretKey == "" {
		return nil, errors.New("Secret key is required")
	}

	redisAddr := os.Getenv("REDIS_HOST")
	if redisAddr == "" {
		return nil, errors.New("Redis is required")
	}

	return &Config{
		DatabaseURL: databaseURL,
		HTTPAddr:    httpAddr,
		SecretKey:   secretKey,
		RedisAddr:   redisAddr,
	}, nil
}
