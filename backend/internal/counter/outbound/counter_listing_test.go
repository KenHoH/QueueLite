package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"context"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"strings"
	"testing"
)

func TestCounterListingQueryIsScopedToBusiness(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost user=test dbname=test sslmode=disable"}), &gorm.Config{DisableAutomaticPing: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	business, other := uuid.New(), uuid.New()
	called := false
	db.Callback().Query().Replace("gorm:query", func(tx *gorm.DB) {
		callbacks.BuildQuerySQL(tx)
		called = true
		sql := tx.Statement.SQL.String()
		vars := tx.Statement.Vars
		if !strings.Contains(sql, "business_id = $1") || !strings.Contains(sql, "ORDER BY name ASC, id ASC") || len(vars) != 1 || vars[0] != business {
			t.Fatalf("unscoped listing: %s %#v", sql, vars)
		}
		*tx.Statement.Dest.(*[]model.Counter) = []model.Counter{{ID: uuid.New(), BusinessID: business, Name: "Desk"}}
	})
	result, err := NewCounterRepo(db).ListBusinessCounters(context.Background(), business)
	if err != nil || !called || len(result) != 1 || result[0].BusinessID != business || result[0].BusinessID == other {
		t.Fatalf("incorrect listing: %+v %v", result, err)
	}
}
