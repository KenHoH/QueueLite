package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/business/domain"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestGetAllBusinessMapsCompleteRecords(t *testing.T) {
	// Replace only the database query boundary: exercise the actual repository
	// mapping and cursor path without requiring a running Postgres instance.
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost user=test dbname=test sslmode=disable"}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	description := "Local appointments"
	createdAt := time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC)
	openTime, _ := time.Parse("15:04", "09:00")
	closeTime, _ := time.Parse("15:04", "21:00")
	records := []model.Business{{ID: uuid.New(), Name: "Northside", Location: "Jakarta", Description: &description, Operational: true, OpenTime: openTime, CloseTime: closeTime, Email: "hello@example.test", PhoneNumber: "628123456789", CreatedAt: createdAt}}
	if err := db.Callback().Query().Replace("gorm:query", func(tx *gorm.DB) {
		*tx.Statement.Dest.(*[]model.Business) = records
	}); err != nil {
		t.Fatal(err)
	}
	repo := NewBusinessRepo(db)
	for _, operational := range []bool{true, false} {
		records[0].Operational = operational
		got, cursor, err := repo.GetAllBusiness(context.Background(), nil, 12)
		if err != nil {
			t.Fatal(err)
		}
		want := []domain.Business{{ID: records[0].ID, Name: "Northside", Location: "Jakarta", Description: &description, Operational: operational, OpenTime: openTime, CloseTime: closeTime, Email: "hello@example.test", PhoneNumber: "628123456789", CreatedAt: createdAt}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("incomplete business fields: %#v", got)
		}
		if cursor == nil || cursor.ID != records[0].ID || !cursor.CreatedAt.Equal(createdAt) {
			t.Fatalf("incorrect cursor: %#v", cursor)
		}
	}
	records = nil
	got, cursor, err := repo.GetAllBusiness(context.Background(), nil, 12)
	if err != nil || len(got) != 0 || cursor != nil {
		t.Fatalf("empty page: %#v, %#v, %v", got, cursor, err)
	}
}
