package store

import (
	"database/sql"
	"testing"
)

func TestOpsInsertRead(t *testing.T) {
	db, _ := Open(":memory:")
	defer db.Close()
	Migrate(db)
	_, err := db.Exec(`INSERT INTO ops(id,device_id,hlc,entity,entity_id,field,value,created_at) VALUES (?,?,?,?,?,?,?,?)`,
		"01", "dev", "hlc", "book", "sha", "title", []byte("x"), "now")
	if err != nil {
		t.Fatal(err)
	}
	var id string
	db.QueryRow("SELECT id FROM ops WHERE seq=1").Scan(&id)
	if id != "01" {
		t.Fatalf("got %s", id)
	}
	_ = sql.Open
}
