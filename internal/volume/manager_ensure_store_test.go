package volume

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_EnsurePnpmStore_CreatePath(t *testing.T) {
	dir := t.TempDir()
	// Fake runtime that returns not-exist for inspect, empty list for ls, success for create
	rt := filepath.Join(dir, "rt.sh")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = volume ] && [ \"$2\" = inspect ]; then exit 1; fi\n" +
		"if [ \"$1\" = volume ] && [ \"$2\" = ls ]; then exit 0; fi\n" +
		"if [ \"$1\" = volume ] && [ \"$2\" = create ]; then exit 0; fi\n" +
		"exit 0\n"
	os.WriteFile(rt, []byte(script), 0o755)

	vm := NewManager(rt, t.TempDir())
	if _, ok := vm.metadata[vm.pnpmStore]; !ok {
		t.Fatalf("expected pnpm store metadata to be created")
	}
}
