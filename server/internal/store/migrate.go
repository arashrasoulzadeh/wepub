package store

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(db *sql.DB) error {
	_, _ = db.Exec("CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)")
	var v int
	row := db.QueryRow("SELECT COALESCE(MAX(version),0) FROM schema_version")
	row.Scan(&v)
	entries, _ := migrationsFS.ReadDir("migrations")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		num := 0
		fmt.Sscanf(name, "%d", &num)
		if num <= v {
			continue
		}
		b, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(b)); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		// record version
		if v == 0 {
			_, _ = db.Exec("INSERT INTO schema_version(version) VALUES (?)", num)
		} else {
			_, _ = db.Exec("UPDATE schema_version SET version=?", num)
		}
		v = num
	}
	// handle multiple statements already executed
	_ = strings.TrimSpace
	return nil
}
