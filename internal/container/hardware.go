// Package container provides hardware detection and optimization hints
package container

import (
	"os/exec"
	"runtime"
	"strings"
)

// DetectAppleSiliconGeneration returns M1, M2, M3, or unknown
func DetectAppleSiliconGeneration() string {
	if runtime.GOOS != osDarwin || runtime.GOARCH != archArm64 {
		return ""
	}
	cmd := execCommand("system_profiler", "SPHardwareDataType")
	output, err := cmd.Output()
	if err != nil {
		return strUnknown
	}
	outputStr := string(output)
	if strings.Contains(outputStr, "Apple M3") {
		return "M3"
	} else if strings.Contains(outputStr, "Apple M2") {
		return "M2"
	} else if strings.Contains(outputStr, "Apple M1") {
		return "M1"
	}
	return strUnknown
}

// OptimizationHints returns platform-specific performance tips
func OptimizationHints() []string {
	hints := []string{}
	if runtime.GOOS == osDarwin && runtime.GOARCH == archArm64 {
		if _, err := exec.LookPath(rtContainer); err != nil {
			hints = append(hints,
				"Install Apple Container for 5-10x speed boost:",
				"Download from developer.apple.com/virtualization",
			)
		}
		dockerCmd := execCommand(rtDocker, "info")
		if output, err := dockerCmd.Output(); err == nil {
			if strings.Contains(string(output), "Docker Desktop") {
				hints = append(hints,
					"Docker Desktop detected. Consider Finch or Podman for better performance on Apple Silicon",
				)
			}
		}
	}
	return hints
}
