package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"context"
	"errors"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"strings"
	"testing"
)

func TestMemberListingSelectsOnlyOperationalFieldsForBusiness(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost user=test dbname=test sslmode=disable"}), &gorm.Config{DisableAutomaticPing: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	business := uuid.New()
	called := false
	db.Callback().Row().After("gorm:row").Register("test:members-scope", func(tx *gorm.DB) {
		called = true
		sql := tx.Statement.SQL.String()
		vars := tx.Statement.Vars
		if !strings.Contains(sql, "relations.business_id = $1") || !strings.Contains(sql, "relations.user_id, users.username, relations.role") || strings.Contains(sql, "email") || strings.Contains(sql, "phone") || len(vars) != 1 || vars[0] != business {
			t.Fatalf("unsafe member query: %s %#v", sql, vars)
		}
	})
	_, err = NewBusinessRepo(db).ListBusinessMembers(context.Background(), business)
	if !called || !errors.Is(err, gorm.ErrDryRunModeUnsupported) {
		t.Fatalf("dry-run member SQL was not inspected: %v", err)
	}
}

func TestMembershipQueriesExcludeOtherAccounts(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost user=test dbname=test sslmode=disable"}), &gorm.Config{DisableAutomaticPing: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	user, business := uuid.New(), uuid.New()
	queries := 0
	db.Callback().Query().Replace("gorm:query", func(tx *gorm.DB) {
		callbacks.BuildQuerySQL(tx)
		queries++
		sql := tx.Statement.SQL.String()
		vars := tx.Statement.Vars
		if !strings.Contains(sql, `"user_business_relations"`) || !strings.Contains(sql, "user_id = $1") || len(vars) == 0 || vars[0] != user {
			t.Fatalf("unscoped relation query: %s %#v", sql, vars)
		}
		if queries == 1 {
			if _, ok := tx.Statement.Preloads["Business"]; !ok {
				t.Fatal("business records are not loaded")
			}
		} else {
			if !strings.Contains(sql, "business_id = $2") || vars[1] != business {
				t.Fatalf("wrong business access: %s %#v", sql, vars)
			}
			tx.Statement.Dest.(*model.UserBusinessRelation).Role = model.BusinessRole("admin")
		}
	})
	items, err := NewBusinessRepo(db).GetUserBusinesses(context.Background(), user)
	if err != nil || items == nil || len(items) != 0 {
		t.Fatalf("empty scoped result: %v %v", items, err)
	}
	role, err := NewBusinessRepo(db).GetUserBusinessRole(context.Background(), user, business)
	if err != nil || role != "admin" || queries != 2 {
		t.Fatalf("scoped role: %s %v", role, err)
	}
}
