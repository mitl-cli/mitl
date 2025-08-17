package container

import (
	"os/exec"
	"testing"
)

func TestGetRuntimeVersion_WithStub(t *testing.T) {
	rm := NewManager()
	prev := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-lc", "printf 'v1.2.3' ")
	}
	defer func() { execCommand = prev }()
	v := rm.getRuntimeVersion("any")
	if v == "" {
		t.Fatalf("expected version string")
	}
}
