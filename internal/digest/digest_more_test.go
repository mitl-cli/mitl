package digest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectTag_InvalidAlgorithm(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)
	_, err := ProjectTag(dir, &Options{Algorithm: "md5"})
	if err == nil {
		t.Fatalf("expected error for invalid algorithm")
	}
}

func TestProjectCalculator_LockfilesOnly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "go.sum"), []byte("a v1 h1:abc"), 0o644)
	calc := NewProjectCalculator(dir, &Options{Algorithm: "sha256", LockfilesOnly: true})
	d, err := calc.Calculate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if d.FileCount < 1 || len(d.Files) < 1 {
		t.Fatalf("expected lockfile to be included only: %+v", d)
	}
}
