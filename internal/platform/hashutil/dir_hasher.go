package hashutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type DirHasher struct {
	Root string
}

func (h DirHasher) Hash(_ context.Context) (string, error) {
	var files []string
	err := filepath.WalkDir(h.Root, func(path string, d os.DirEntry, err error) error {
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
	sum := sha256.New()
	for _, file := range files {
		if _, err := io.WriteString(sum, file); err != nil {
			return "", err
		}
		content, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		if _, err := sum.Write(content); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}
