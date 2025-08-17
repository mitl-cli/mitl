package commands

import (
	"os"
	"os/exec"
	"testing"
)

func TestShell_EchoRun(t *testing.T) {
	tmp := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", oldHome)

	os.Setenv("MITL_RUN_CLI", "/bin/echo")
	old := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd { return exec.Command("sh", "-c", "true") }
	defer func() { execCommand = old }()

	if err := Shell(nil); err != nil {
		t.Fatalf("shell: %v", err)
	}
}
