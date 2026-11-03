package store

import (
	"database/sql"
	"testing"
)

func TestOpenPragmas(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var jm string
	db.QueryRow("PRAGMA journal_mode").Scan(&jm)
	// in-memory may report memory; check foreign_keys and busy_timeout
	var fk int
	db.QueryRow("PRAGMA foreign_keys").Scan(&fk)
	if fk != 1 {
		t.Fatalf("foreign_keys=%d", fk)
	}
	var bt int
	db.QueryRow("PRAGMA busy_timeout").Scan(&bt)
	if bt != 5000 {
		t.Fatalf("busy_timeout=%d", bt)
	}
	_ = sql.Open
}
