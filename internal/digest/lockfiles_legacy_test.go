package digest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockfileHasher_NpmLegacyDependencies(t *testing.T) {
	dir := t.TempDir()
	// Legacy package-lock.json with top-level dependencies tree
	content := `{
  "name": "app",
  "lockfileVersion": 1,
  "dependencies": {
    "a": {"version":"1.0.0", "dependencies": {"b": {"version":"2.0.0"}}}
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	h := NewLockfileHasher(dir)
	sum, err := h.HashLockfiles()
	if err != nil || sum == "no-lockfiles" || sum == "" {
		t.Fatalf("unexpected hash: %q err=%v", sum, err)
	}
}
