package digest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockfileHasher_ComposerLock(t *testing.T) {
	dir := t.TempDir()
	data := `{"packages":[{"name":"vendor/pkg","version":"1.0.0"}],"content-hash":"abc123"}`
	if err := os.WriteFile(filepath.Join(dir, "composer.lock"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	h := NewLockfileHasher(dir)
	sum, err := h.HashLockfiles()
	if err != nil || sum == "no-lockfiles" || sum == "" {
		t.Fatalf("unexpected: %q %v", sum, err)
	}
}

func TestLockfileHasher_PnpmLock(t *testing.T) {
	dir := t.TempDir()
	yaml := "lockfileVersion: '9'\npackages:\n  /a/1.0.0:\n    resolution: {integrity: sha512-abc}\n"
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	h := NewLockfileHasher(dir)
	sum, err := h.HashLockfiles()
	if err != nil || sum == "no-lockfiles" || sum == "" {
		t.Fatalf("unexpected: %q %v", sum, err)
	}
}

func TestLockfileHasher_GemfileLock(t *testing.T) {
	dir := t.TempDir()
	gem := "GEM\n  specs:\n    rake (13.0.6)\n"
	if err := os.WriteFile(filepath.Join(dir, "Gemfile.lock"), []byte(gem), 0o644); err != nil {
		t.Fatal(err)
	}
	h := NewLockfileHasher(dir)
	sum, err := h.HashLockfiles()
	if err != nil || sum == "no-lockfiles" || sum == "" {
		t.Fatalf("unexpected: %q %v", sum, err)
	}
}

func TestLockfileHasher_PythonReqsAndPoetry(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask==2.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "poetry.lock"), []byte("name = \"pkg\"\nversion = \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := NewLockfileHasher(dir)
	sum, err := h.HashLockfiles()
	if err != nil || sum == "no-lockfiles" || sum == "" {
		t.Fatalf("unexpected: %q %v", sum, err)
	}
}

func TestLockfileHasher_PipfileAndCargo(t *testing.T) {
	dir := t.TempDir()
	pip := `{"default": {"requests": {"version": "==2.28.0"}}}`
	if err := os.WriteFile(filepath.Join(dir, "Pipfile.lock"), []byte(pip), 0o644); err != nil {
		t.Fatal(err)
	}
	cargo := "name = \"rand\"\nversion = \"0.8.5\"\n"
	if err := os.WriteFile(filepath.Join(dir, "Cargo.lock"), []byte(cargo), 0o644); err != nil {
		t.Fatal(err)
	}
	h := NewLockfileHasher(dir)
	sum, err := h.HashLockfiles()
	if err != nil || sum == "no-lockfiles" || sum == "" {
		t.Fatalf("unexpected: %q %v", sum, err)
	}
}
