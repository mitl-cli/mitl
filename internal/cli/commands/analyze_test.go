package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyze_WithStubbedBinaries(t *testing.T) {
	dir := t.TempDir()
	// Create stub binaries that just echo arguments
	for _, name := range []string{"php", "node", "pnpm"} {
		p := filepath.Join(dir, name)
		if err := os.Symlink("/bin/echo", p); err != nil {
			// Fallback: write shell script
			if err2 := os.WriteFile(p, []byte("#!/bin/sh\necho $0 $@\n"), 0755); err2 != nil {
				t.Skip("cannot create stubs")
			}
		}
	}
	old := os.Getenv("PATH")
	os.Setenv("PATH", dir+":"+old)
	defer os.Setenv("PATH", old)

	if err := Analyze(nil); err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
}

func TestRunCommandEcho(t *testing.T) {
	out := runCommand("/bin/echo", "hello")
	if out == "" {
		t.Fatalf("expected output from echo")
	}
}
