package blob

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Put(root, sha string, r io.Reader) error {
	p, err := Path(root, sha)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), r); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()
	got := fmt.Sprintf("%x", h.Sum(nil))
	if got != sha {
		os.Remove(tmp)
		return fmt.Errorf("hash mismatch")
	}
	return os.Rename(tmp, p)
}
