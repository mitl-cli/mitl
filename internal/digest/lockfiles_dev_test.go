package digest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockfileHasher_PipfileDevelop(t *testing.T) {
	dir := t.TempDir()
	pip := `{"develop": {"pytest": {"version": "==7.0.0"}}}`
	if err := os.WriteFile(filepath.Join(dir, "Pipfile.lock"), []byte(pip), 0644); err != nil {
		t.Fatal(err)
	}
	h := NewLockfileHasher(dir)
	sum, err := h.HashLockfiles()
	if err != nil || sum == "no-lockfiles" || sum == "" {
		t.Fatalf("unexpected: %q %v", sum, err)
	}
}
