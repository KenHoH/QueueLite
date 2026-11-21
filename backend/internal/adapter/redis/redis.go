package cache

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisOptions struct {
	Addr     string
	Password string
	DB       int
	UseTLS   bool
}

func NewConnectionRedis(redisURL string, ctx context.Context) (*redis.Client, error) {
	return NewConnectionRedisWithOptions(ctx, RedisOptions{Addr: redisURL})
}

func NewConnectionRedisWithOptions(ctx context.Context, opts RedisOptions) (*redis.Client, error) {
	redisOpts := &redis.Options{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	}
	if opts.UseTLS {
		redisOpts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("Failed to connect to Redis: %v", err)
	}
	return rdb, nil
}
