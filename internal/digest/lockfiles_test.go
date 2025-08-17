package digest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockfileHasher_GoSum(t *testing.T) {
	dir := t.TempDir()
	data := []byte("github.com/pkg/errors v0.9.1 h1:abcdef\n")
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	h := NewLockfileHasher(dir)
	sum, err := h.HashLockfiles()
	if err != nil {
		t.Fatalf("HashLockfiles: %v", err)
	}
	if sum == "no-lockfiles" || sum == "" {
		t.Fatalf("unexpected hash: %q", sum)
	}
}
