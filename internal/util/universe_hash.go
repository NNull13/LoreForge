package util

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func HashDir(root string) (string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	h := sha1.New() // #nosec G401: non-crypto identity hash is enough for versioning.
	for _, p := range files {
		if _, err := io.WriteString(h, p); err != nil {
			return "", err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		if _, err := h.Write(b); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
