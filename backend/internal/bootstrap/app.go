package bootstrap

import (
	"QueueLite/internal/adapter/postgres"
	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/config"
	"context"
	"fmt"
	"net/http"
)

var ctx = context.Background()

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := postgres.NewConnection(cfg.DatabaseURL)
	if err != nil {
		return err
	}

	rdb, err := cache.NewConnectionRedis(cfg.RedisAddr, ctx)
	if err != nil {
		return err
	}

	if err := postgres.MigrateDatabase(db); err != nil {
		return err
	}

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: NewRouter(db, rdb),
	}

	return server.ListenAndServe()
}
