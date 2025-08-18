package digest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDigest_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "d.json")
	os.WriteFile(p, []byte("not-json"), 0o644)
	if _, err := LoadDigest(p); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}
