package digest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectTag_Basic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	tag, err := ProjectTag(dir, Options{Algorithm: "sha256"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tag) != 12 {
		t.Fatalf("expected 12-char tag, got %q", tag)
	}
}
