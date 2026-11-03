package store

import (
	"database/sql"
	"testing"
)

func TestMigrateIdempotent(t *testing.T) {
	db, _ := Open(":memory:")
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal("rerun failed")
	}
	var v int
	db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&v)
	if v < 1 {
		t.Fatalf("version=%d", v)
	}
	_ = sql.Open
}
