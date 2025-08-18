package digest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockfileHasher_GoMod(t *testing.T) {
	dir := t.TempDir()
	gomod := `module example.com/app

go 1.21

require (
    github.com/pkg/errors v0.9.1
    golang.org/x/text v0.14.0
)`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}
	h := NewLockfileHasher(dir)
	sum, err := h.HashLockfiles()
	if err != nil || sum == "no-lockfiles" || sum == "" {
		t.Fatalf("unexpected: %q %v", sum, err)
	}
}
