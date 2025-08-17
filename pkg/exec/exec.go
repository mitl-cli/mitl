// Package exec provides command execution utilities and CLI detection
// for the mitl tool. This package centralizes command execution logic
// and provides test-friendly interfaces for mocking.
package exec

import (
	"os"
	"os/exec"

	"mitl/internal/container"
)

// Commander provides an interface for command execution that can be mocked in tests.
// This enables dependency injection and makes code more testable.
type Commander interface {
	Command(name string, args ...string) *exec.Cmd
}

// DefaultCommander implements Commander using the standard exec.Command.
type DefaultCommander struct{}

// Command creates a new exec.Cmd using the standard library exec.Command.
func (DefaultCommander) Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// Global instance that can be overridden in tests
var Default Commander = DefaultCommander{}

// Command is a convenience function that delegates to the global Commander instance.
// Tests can override Default to provide mock implementations.
func Command(name string, args ...string) *exec.Cmd {
	return Default.Command(name, args...)
}

// FindBuildCLI attempts to locate a suitable container build CLI. The
// priority order is influenced by the host OS. On macOS, the native
// `container` CLI (from Apple's Containerization framework) is preferred
// when present, followed by finch, podman and nerdctl. On other OSes,
// podman and nerdctl are preferred. Docker is always used as the last
// resort. Environment variables and user configuration override the
// auto‑detection logic.
func FindBuildCLI() string {
	// Environment variable takes highest priority
	if env := os.Getenv("MITL_BUILD_CLI"); env != "" {
		if _, err := exec.LookPath(env); err == nil {
			return env
		}
	}
	// TODO: Add configuration file support - needs refactoring to avoid circular imports
	// Intelligent selection
	rm := container.NewManager()
	return rm.SelectOptimal()
}

// FindRunCLI attempts to locate a suitable container run CLI. The logic
// mirrors FindBuildCLI but allows override via MITL_RUN_CLI. In practice,
// the same binary can be used for building and running, but having two
// separate functions allows for future differences in behaviour if needed.
func FindRunCLI() string {
	// Environment variable takes highest priority
	if env := os.Getenv("MITL_RUN_CLI"); env != "" {
		if _, err := exec.LookPath(env); err == nil {
			return env
		}
	}
	// TODO: Add configuration file support - needs refactoring to avoid circular imports
	// Intelligent selection (use same runtime for consistency)
	rm := container.NewManager()
	return rm.SelectOptimal()
}
