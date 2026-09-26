package cache

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func NewConnectionRedis(redisURL string, ctx context.Context) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: "", // No password by default
		DB:       0,  // Default DB
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
		return nil, err
	}
	return rdb, nil
}
