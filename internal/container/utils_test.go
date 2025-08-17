package container

import (
	"os/exec"
	"testing"
)

func TestFormatRuntime(t *testing.T) {
	rt := Runtime{Name: "docker", Version: "24.0.5", Capabilities: []string{"buildkit", "compose"}, Performance: 1.5}
	s := FormatRuntime(rt)
	if s == "" || s == rt.Name {
		t.Fatalf("unexpected format: %q", s)
	}
}

func TestIsOptimalRuntime(t *testing.T) {
	m := &Manager{}
	m.hardwareProfile.IsAppleSilicon = true
	if !m.IsOptimalRuntime("container") {
		t.Errorf("expected container optimal on Apple Silicon")
	}
	m.hardwareProfile.IsAppleSilicon = false
	if !m.IsOptimalRuntime("docker") {
		t.Errorf("expected docker optimal on non-Apple Silicon")
	}
}

func TestDetectCapabilities_WithStubbedExec(t *testing.T) {
	// Stub execCommand to always succeed, regardless of args
	old := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 0")
	}
	defer func() { execCommand = old }()

	m := &Manager{}
	caps := m.detectCapabilities("docker")
	if len(caps) == 0 {
		t.Errorf("expected some capabilities when commands succeed")
	}
}
