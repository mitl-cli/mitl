package container

import (
	"fmt"
	"strings"
)

// getRuntimeVersion extracts the version string for a runtime
func (m *Manager) getRuntimeVersion(name string) string {
	cmd := execCommand(name, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		// Extract version from first line
		parts := strings.Fields(lines[0])
		for i, p := range parts {
			if strings.Contains(p, ".") && i > 0 {
				return p
			}
		}
	}
	return "unknown"
}

// detectCapabilities determines what features a runtime supports
func (m *Manager) detectCapabilities(name string) []string {
	caps := []string{}

	// Check for BuildKit support
	if name == "docker" || name == "container" {
		cmd := execCommand(name, "buildx", "version")
		if err := cmd.Run(); err == nil {
			caps = append(caps, "buildkit")
		}
	}

	// Check for multi-platform support
	if name != "finch" { // Finch doesn't support multi-platform well
		caps = append(caps, "multi-platform")
	}

	// Check for compose support
	cmd := execCommand(name, "compose", "version")
	if err := cmd.Run(); err == nil {
		caps = append(caps, "compose")
	}

	return caps
}

// FormatRuntime returns a formatted string for runtime display
func FormatRuntime(rt Runtime) string {
	var features []string
	if rt.Performance > 0 {
		features = append(features, fmt.Sprintf("%.1fx", rt.Performance))
	}
	if len(rt.Capabilities) > 0 {
		features = append(features, strings.Join(rt.Capabilities, ", "))
	}

	if len(features) > 0 {
		return fmt.Sprintf("%s %s (%s)", rt.Name, rt.Version, strings.Join(features, ", "))
	}
	return fmt.Sprintf("%s %s", rt.Name, rt.Version)
}

// IsOptimalRuntime checks if the given runtime is optimal for the hardware
func (m *Manager) IsOptimalRuntime(name string) bool {
	if m.hardwareProfile.IsAppleSilicon && name == "container" {
		return true
	}
	if !m.hardwareProfile.IsAppleSilicon && (name == "podman" || name == "docker") {
		return true
	}
	return false
}
