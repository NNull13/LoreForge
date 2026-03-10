package hash_util

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDirHasherHashIsDeterministic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("second"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "a.txt"), []byte("first"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	hasher := DirHasher{Root: root}
	hashA, err := hasher.Hash(context.Background())
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	hashB, err := hasher.Hash(context.Background())
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	if hashA == "" || hashA != hashB {
		t.Fatalf("unexpected hashes: %q %q", hashA, hashB)
	}
}

func TestDirHasherHashErrorsForMissingRoot(t *testing.T) {
	t.Parallel()

	if _, err := (DirHasher{Root: filepath.Join(t.TempDir(), "missing")}).Hash(context.Background()); err == nil {
		t.Fatal("expected missing root error")
	}
}
