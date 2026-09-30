package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/queue/app"
	"QueueLite/internal/queue/domain"
	"context"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"testing"
)

func TestStreamAndRepositoryPreserveUUID(t *testing.T) {
	id, business, owner := uuid.New(), uuid.New(), uuid.New()
	q, err := app.QueueFromStreamValues(map[string]any{"id": id.String(), "business_id": business.String(), "user_id": owner.String(), "name": "A001", "priority": "false"})
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost user=test dbname=test sslmode=disable"}), &gorm.Config{DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	var records []model.Queue
	if err := db.Callback().Create().Replace("gorm:create", func(tx *gorm.DB) {
		switch dest := tx.Statement.Dest.(type) {
		case *model.Queue:
			records = append(records, *dest)
		case *[]model.Queue:
			records = append(records, (*dest)...)
		default:
			t.Fatalf("unexpected create: %T", dest)
		}
	}); err != nil {
		t.Fatal(err)
	}
	repo := NewQueueRepo(db)
	got, err := repo.CreateQueue(context.Background(), q)
	if err != nil || got.ID != id {
		t.Fatalf("single UUID changed: %v %v", got, err)
	}
	if err := repo.CreateQueues(context.Background(), []domain.Queue{*q}); err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatal("insert callbacks missing")
	}
	for _, record := range records {
		if record.ID != id || record.UserID == nil || *record.UserID != owner || record.BusinessID != business {
			t.Fatalf("model UUID changed: %#v", record)
		}
	}
}
