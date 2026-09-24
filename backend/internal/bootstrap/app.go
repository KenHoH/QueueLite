package bootstrap

import (
	"QueueLite/internal/adapter/postgres"
	"QueueLite/internal/config"
	"fmt"
	"net/http"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := postgres.NewConnection(cfg.DatabaseURL)
	if err != nil {
		return err
	}

	if err := postgres.MigrateDatabase(db); err != nil {
		return err
	}

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: NewRouter(db),
	}

	return server.ListenAndServe()
}
