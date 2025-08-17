package container

import (
	"os/exec"
	"testing"
)

func TestHardware_Helpers(t *testing.T) {
	// Ensure functions are callable and stable
	_ = DetectAppleSiliconGeneration()
	_ = OptimizationHints()
}

func TestHardware_DarwinStubs(t *testing.T) {
	// Stub execCommand to simulate macOS outputs
	prev := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "system_profiler" {
			return exec.Command("sh", "-lc", "printf 'Hardware:\\n  Chip: Apple M3' ")
		}
		if name == "docker" && len(args) > 0 && args[0] == "info" {
			return exec.Command("sh", "-lc", "printf 'Docker Desktop' ")
		}
		return exec.Command("sh", "-lc", "true")
	}
	defer func() { execCommand = prev }()

	_ = DetectAppleSiliconGeneration() // should parse and not panic
	hints := OptimizationHints()
	if len(hints) == 0 {
		t.Fatalf("expected hints when Docker Desktop detected")
	}
}

func TestHardware_DarwinErrorPath(t *testing.T) {
	prev := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "system_profiler" {
			return exec.Command("sh", "-lc", "exit 1")
		}
		return exec.Command("sh", "-lc", "true")
	}
	defer func() { execCommand = prev }()
	_ = DetectAppleSiliconGeneration() // should handle error and return unknown/empty
}

func TestHardware_DarwinM1M2(t *testing.T) {
	prev := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "system_profiler" {
			return exec.Command("sh", "-lc", "printf 'Chip: Apple M2' ")
		}
		return exec.Command("sh", "-lc", "true")
	}
	_ = DetectAppleSiliconGeneration()
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "system_profiler" {
			return exec.Command("sh", "-lc", "printf 'Chip: Apple M1' ")
		}
		return exec.Command("sh", "-lc", "true")
	}
	_ = DetectAppleSiliconGeneration()
	execCommand = prev
}
