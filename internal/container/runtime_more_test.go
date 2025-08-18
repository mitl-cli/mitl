package container

import (
	"os/exec"
	"testing"
)

func TestManager_imageAvailable(t *testing.T) {
	m := NewManager()
	old := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		// Return non-empty output to indicate image exists
		return exec.Command("sh", "-c", "echo id123")
	}
	defer func() { execCommand = old }()
    rt := Runtime{Name: "docker", Path: "/bin/echo"}
    if !m.imageAvailable(&rt, "alpine:latest") {
        t.Fatalf("expected imageAvailable true")
    }
}

func TestGetRuntimeVersion_Parsing(t *testing.T) {
	m := NewManager()
	old := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'docker version 24.0.5, build abc' && true")
	}
	defer func() { execCommand = old }()
	v := m.getRuntimeVersion("docker")
	if v == "unknown" {
		t.Fatalf("expected parsed version, got %q", v)
	}
}
