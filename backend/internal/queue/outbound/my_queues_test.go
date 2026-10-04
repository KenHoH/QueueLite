package outbound

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
)

func TestActiveUserQueuesQueryScopesOwnerAndAllActiveStates(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost user=test dbname=test sslmode=disable"}), &gorm.Config{DisableAutomaticPing: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	owner := uuid.New()
	checked := false
	err = db.Callback().Query().Replace("gorm:query", func(tx *gorm.DB) {
		callbacks.BuildQuerySQL(tx)
		sql := tx.Statement.SQL.String()
		if !strings.Contains(sql, "user_id = $1 AND state IN ($2,$3,$4)") {
			t.Fatalf("unscoped query: %s", sql)
		}
		if !strings.Contains(sql, "ORDER BY created_at ASC, id ASC") {
			t.Fatalf("unstable ordering: %s", sql)
		}
		vars := tx.Statement.Vars
		if len(vars) != 4 || vars[0] != owner {
			t.Fatalf("wrong owner: %#v", vars)
		}
		for i, state := range []string{"waiting", "called", "processing"} {
			if fmt.Sprint(vars[i+1]) != state {
				t.Fatalf("wrong active state: %#v", vars)
			}
		}
		checked = true
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := NewQueueRepo(db).GetActiveQueuesByUser(context.Background(), owner)
	if err != nil || len(got) != 0 || !checked {
		t.Fatalf("query not exercised: %v %v", got, err)
	}
}
