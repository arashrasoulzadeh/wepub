package blob

import (
	"os"
	"path/filepath"
	"strings"
)

type Info struct {
	Hash string
	Size int64
	MTime int64
}

func List(root string) ([]Info, error) {
	var out []Info
	base := filepath.Join(root, "blobs")
	filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() { return nil }
		// only 64 hex filenames
		name := filepath.Base(p)
		if len(name) != 64 { return nil }
		out = append(out, Info{Hash: name, Size: info.Size(), MTime: info.ModTime().Unix()})
		return nil
	})
	_ = strings.ToLower
	return out, nil
}
