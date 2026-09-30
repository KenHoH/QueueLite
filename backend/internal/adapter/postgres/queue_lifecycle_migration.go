package postgres

import (
	"QueueLite/internal/adapter/postgres/model"
	"fmt"
	"gorm.io/gorm"
)

// MigrateQueueLifecycle adds only missing nullable lifecycle columns using the
// repository's existing GORM migration mechanism. Run explicitly against a
// reviewed database; startup does not invoke migrations. No historical backfill.
func MigrateQueueLifecycle(db *gorm.DB) error {
	for _, field := range []string{"CalledAt", "ProcessingAt", "DoneAt", "CancelledAt"} {
		if !db.Migrator().HasColumn(&model.Queue{}, field) {
			if err := db.Migrator().AddColumn(&model.Queue{}, field); err != nil {
				return fmt.Errorf("migrate queue lifecycle %s: %w", field, err)
			}
		}
	}
	return nil
}
