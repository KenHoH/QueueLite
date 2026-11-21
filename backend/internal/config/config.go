package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var dotenvOnce sync.Once

type Config struct {
	DatabaseURL      string
	HTTPAddr         string
	SecretKey        string
	RedisAddr        string
	RedisPassword    string
	RedisDB          int
	RedisTLS         bool
	TelebotAPI       string
	CookieSecure     bool
	PublicBaseURL    string
	QueueWorkerCount int
	DBMaxOpenConns   int
	DBMaxIdleConns   int
}

const (
	QueueExpirationTime                  = 24 * time.Hour
	GuestExpirationTime                  = 24 * time.Hour
	StreamName                    string = "queue_stream"
	QueueConsumerGroup            string = "queue_stream_workers"
	QueueConsumerName             string = "queue-worker-1"
	QueueDeadLetterStreamName     string = "queue_stream_dead_letter"
	QueueStreamMaxLen             int64  = 50000
	TelebotNotificationStreamName string = "telebot_notification_stream"
	TelebotConsumerGroup          string = "telebot_notification_workers"
	TelebotConsumerName           string = "telebot-worker-1"
	TelebotDeadLetterStreamName   string = "telebot_notification_dead_letter"
)

func loadDotEnvOnce() {
	dotenvOnce.Do(func() {
		_ = godotenv.Load()
	})
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}

func Load() (*Config, error) {
	loadDotEnvOnce()

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

	redisDB := 0
	if raw := strings.TrimSpace(os.Getenv("REDIS_DB")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			redisDB = parsed
		}
	}

	return &Config{
		DatabaseURL:      databaseURL,
		HTTPAddr:         httpAddr,
		SecretKey:        secretKey,
		RedisAddr:        redisAddr,
		RedisPassword:    os.Getenv("REDIS_PASSWORD"),
		RedisDB:          redisDB,
		RedisTLS:         envBool("REDIS_TLS", false),
		TelebotAPI:       os.Getenv("TELEBOT_API_KEY"),
		CookieSecure:     envBool("COOKIE_SECURE", false),
		PublicBaseURL:    strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/"),
		QueueWorkerCount: envInt("QUEUE_WORKER_COUNT", 2),
		DBMaxOpenConns:   envInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:   envInt("DB_MAX_IDLE_CONNS", 10),
	}, nil
}

func LoadTelebotAPIKey() string {
	loadDotEnvOnce()
	return os.Getenv("TELEBOT_API_KEY")
}

func IsCookieSecure() bool {
	loadDotEnvOnce()
	return envBool("COOKIE_SECURE", false)
}

func PublicBaseURL() string {
	loadDotEnvOnce()
	return strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
}
