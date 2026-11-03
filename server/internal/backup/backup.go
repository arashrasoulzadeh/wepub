package backup

import (
	"database/sql"
	"fmt"
	"os"
)

func SQLiteBackup(db *sql.DB, dst string) error {
	// VACUUM INTO requires target not exist
	os.Remove(dst)
	_, err := db.Exec(fmt.Sprintf("VACUUM INTO '%s'", dst))
	return err
}
