package blob

import (
	"net/http"
	"os"
)

func Serve(w http.ResponseWriter, r *http.Request, root, sha string) {
	p, err := Path(root, sha)
	if err != nil {
		http.Error(w, "bad sha", http.StatusBadRequest)
		return
	}
	f, err := os.Open(p)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	fi, _ := f.Stat()
	w.Header().Set("ETag", `"`+sha+`"`)
	w.Header().Set("Cache-Control", "immutable")
	http.ServeContent(w, r, sha, fi.ModTime(), f)
}
