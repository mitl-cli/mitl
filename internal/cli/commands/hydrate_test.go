package commands

import (
	"os"
	"os/exec"
	"testing"
)

func TestHydrate_EchoBuild(t *testing.T) {
	tmp := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", oldHome)

	// Force build CLI and stub exec
	os.Setenv("MITL_BUILD_CLI", "/bin/echo")
	old := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd { return exec.Command("sh", "-c", "true") }
	defer func() { execCommand = old }()

	if err := Hydrate(nil); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
}
