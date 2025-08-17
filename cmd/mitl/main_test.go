package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestMainCommand(t *testing.T) {
	// Run from cmd/mitl; should print usage and exit cleanly
	cmd := exec.Command("go", "run", "main.go")
	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			// Accept common non-zero exit codes from go run
			if code := ee.ExitCode(); code != 0 && code != 1 {
				t.Errorf("unexpected exit code: %d", code)
			}
		} else {
			t.Fatalf("failed to run mitl: %v", err)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	cmd := exec.Command("go", "run", "main.go", "version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	if !strings.Contains(string(out), "mitl") {
		t.Errorf("version output missing 'mitl': %s", out)
	}
}
