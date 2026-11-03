package watch

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/arashrasoulzadeh/wepub/server/internal/ingest"
)

func ScanOnce(dir string, process func(string) error) (int, []error) {
	entries, err := os.ReadDir(dir)
	if err != nil { return 0, []error{err} }
	count := 0
	var errs []error
	for _, e := range entries {
		if e.IsDir() { continue }
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".epub") { continue }
		p := filepath.Join(dir, e.Name())
		zr, err := ingest.OpenEPUB(p)
		if err != nil { errs = append(errs, err); continue }
		zr.Close()
		if err := process(p); err != nil { errs = append(errs, err); continue }
		count++
		// move to .done
		os.MkdirAll(filepath.Join(dir, ".done"), 0755)
		os.Rename(p, filepath.Join(dir, ".done", e.Name()))
	}
	return count, errs
}
