package ingest

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
)

const maxEntries = 10000
const maxSize = 500 * 1024 * 1024

func OpenEPUB(path string) (*zip.ReadCloser, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	if len(zr.File) > maxEntries {
		zr.Close()
		return nil, fmt.Errorf("too many entries")
	}
	var total uint64
	for _, f := range zr.File {
		if stringsHasDotDot(f.Name) {
			zr.Close()
			return nil, fmt.Errorf("zip slip")
		}
		total += f.UncompressedSize64
		if total > maxSize {
			zr.Close()
			return nil, fmt.Errorf("too large")
		}
	}
	return zr, nil
}

func stringsHasDotDot(s string) bool {
	// simple check
	for i := 0; i+2 < len(s); i++ {
		if s[i] == '.' && s[i+1] == '.' {
			// ensure path separator context
			return true
		}
	}
	_ = io.EOF
	_ = os.ErrNotExist
	return false
}
