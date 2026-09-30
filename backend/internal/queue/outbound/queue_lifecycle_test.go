package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/queue/domain"
	"context"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/schema"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestQueueLifecycleMapsAndPersistsNullableColumns(t *testing.T) {
	now := time.Now().UTC()
	called, processing, done, cancelled := now, now.Add(time.Minute), now.Add(2*time.Minute), now.Add(3*time.Minute)
	q := domain.Queue{ID: uuid.New(), BusinessID: uuid.New(), Name: "A001", State: domain.QueueStateCancelled, CalledAt: &called, ProcessingAt: &processing, DoneAt: &done, CancelledAt: &cancelled}
	record := toQueueRecord(&q)
	if !reflect.DeepEqual(toDomainQueue(&record), &q) {
		t.Fatal("lifecycle timestamps lost in mapping")
	}
	empty := toQueueRecord(&domain.Queue{})
	if toDomainQueue(&empty).CalledAt != nil || toDomainQueue(&empty).CancelledAt != nil {
		t.Fatal("fabricated historical timestamps")
	}
	parsed, err := schema.Parse(&model.Queue{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"called_at", "processing_at", "done_at", "cancelled_at"} {
		field := parsed.FieldsByDBName[column]
		if field == nil || field.NotNull || field.DataType != "timestamptz" {
			t.Fatalf("missing/non-null lifecycle column: %s", column)
		}
	}
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost user=test dbname=test sslmode=disable"}), &gorm.Config{DisableAutomaticPing: true, DryRun: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	checked := false
	db.Callback().Update().Replace("gorm:update", func(tx *gorm.DB) {
		callbacks.Update(&callbacks.Config{})(tx)
		sql := tx.Statement.SQL.String()
		for _, column := range []string{"called_at", "processing_at", "done_at", "cancelled_at"} {
			if !strings.Contains(sql, `"`+column+`"`) {
				t.Fatalf("timestamp missing from SQL: %s", sql)
			}
		}
		updates := tx.Statement.Dest.(map[string]any)
		if !reflect.DeepEqual(updates["called_at"], q.CalledAt) || !reflect.DeepEqual(updates["cancelled_at"], q.CancelledAt) {
			t.Fatal("timestamp update values lost")
		}
		tx.RowsAffected = 1
		checked = true
	})
	if err := NewQueueRepo(db).UpdateQueue(context.Background(), &q); err != nil || !checked {
		t.Fatalf("persistence not exercised: %v", err)
	}
}
