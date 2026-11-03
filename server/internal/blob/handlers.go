package blob

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/arashrasoulzadeh/wepub/server/internal/store"
)

func UploadHandler(root string, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sha := filepath.Base(r.URL.Path)
		if !IsValid(sha) {
			http.Error(w, "bad sha", http.StatusBadRequest)
			return
		}
		if err := Put(root, sha, r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		// record blob row
		fi, _ := os.Stat(mustPath(root, sha))
		store.InsertBlob(db, sha, fi.Size(), "application/octet-stream")
		w.WriteHeader(http.StatusOK)
	}
}

func DownloadHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sha := filepath.Base(r.URL.Path)
		Serve(w, r, root, sha)
	}
}

func mustPath(root, sha string) string {
	p, _ := Path(root, sha)
	return p
}

func ListHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// minimal: not implemented fully
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "[]")
	}
}

func init() {
	_ = io.Discard
}
