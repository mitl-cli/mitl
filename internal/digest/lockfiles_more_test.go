package digest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockfileHasher_NpmV2Packages(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "name": "app",
  "lockfileVersion": 2,
  "packages": {
    "": {"name":"app","version":"1.0.0"},
    "node_modules/a": {"version":"1.2.3"},
    "node_modules/b": {"version":"2.0.0"}
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

func TestLockfileHasher_YarnLock(t *testing.T) {
	dir := t.TempDir()
	yarn := `# yarn lockfile v1
"left-pad@^1.0.0":
  version "1.3.0"
`
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(yarn), 0644); err != nil {
		t.Fatal(err)
	}
	h := NewLockfileHasher(dir)
	sum, err := h.HashLockfiles()
	if err != nil || sum == "no-lockfiles" || sum == "" {
		t.Fatalf("unexpected hash: %q err=%v", sum, err)
	}
}
