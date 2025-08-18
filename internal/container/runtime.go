// Package container provides container runtime detection and selection.
// This file makes Mitl faster on Apple Silicon by preferring the native
// Apple Container runtime and caching benchmark-driven selections.
package container

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Manager handles intelligent selection of container runtimes
// based on performance characteristics and hardware capabilities
type Manager struct {
	mu                sync.RWMutex
	availableRuntimes []Runtime
	benchmarkCache    map[string]BenchmarkResult
	hardwareProfile   HardwareProfile
	configPath        string
}

// Runtime represents a container runtime with its capabilities
type Runtime struct {
	Name          string    `json:"name"`         // docker, podman, finch, container
	Path          string    `json:"path"`         // /usr/local/bin/docker
	Version       string    `json:"version"`      // 24.0.5
	Priority      int       `json:"priority"`     // 1-100, higher = preferred
	Capabilities  []string  `json:"capabilities"` // buildkit, multi-arch, etc
	Performance   float64   `json:"performance"`  // Relative to fastest (1.0 = fastest)
	LastBenchmark time.Time `json:"last_benchmark"`
}

// HardwareProfile describes the host system
type HardwareProfile struct {
	OS             string `json:"os"`   // darwin, linux
	Arch           string `json:"arch"` // arm64, amd64
	IsAppleSilicon bool   `json:"apple_silicon"`
	CPUCores       int    `json:"cpu_cores"`
	MemoryGB       int    `json:"memory_gb"`
}

// BenchmarkResult stores performance test results
type BenchmarkResult struct {
	Runtime   string        `json:"runtime"`
	BuildTime time.Duration `json:"build_time"`
	StartTime time.Duration `json:"start_time"`
	ExecTime  time.Duration `json:"exec_time"`
	// Score prior to normalization: lower is better (composite seconds)
	Score     float64   `json:"score"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
	Mode      string    `json:"mode,omitempty"` // "exec" or "build+exec"
}

type benchmarkCacheFile struct {
	Hardware HardwareProfile            `json:"hardware"`
	Results  map[string]BenchmarkResult `json:"results"`
}

const benchTTL = 14 * 24 * time.Hour

// testable exec command wrapper
var execCommand = exec.Command

// NewManager constructs and initializes a runtime manager
func NewManager() *Manager {
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.Getwd()
	}
	dir := filepath.Join(home, ".mitl")
	_ = os.MkdirAll(dir, 0o755)
	rm := &Manager{
		configPath:     filepath.Join(dir, "benchmarks.json"),
		benchmarkCache: make(map[string]BenchmarkResult),
	}
	rm.detectHardware()
	rm.discoverRuntimes()
	rm.loadBenchmarkCache()
	// Apply cached performance to available runtimes
	rm.applyCachedPerformance()
	return rm
}

// detectHardware fills the hardware profile with OS/arch and Apple Silicon checks
func (rm *Manager) detectHardware() {
	rm.hardwareProfile = HardwareProfile{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	rm.hardwareProfile.IsAppleSilicon = runtime.GOOS == osDarwin && runtime.GOARCH == archArm64

	// Detect Rosetta translation; if translated, treat as non-AppleSilicon for perf purposes
	if rm.hardwareProfile.IsAppleSilicon {
		cmd := execCommand("sysctl", "-n", "sysctl.proc_translated")
		if output, err := cmd.Output(); err == nil {
			if strings.TrimSpace(string(output)) == "1" {
				rm.hardwareProfile.IsAppleSilicon = false
			}
		}
	}
}

// discoverRuntimes probes PATH for candidate runtimes in priority order
func (rm *Manager) discoverRuntimes() {
	candidates := rm.getCandidateRuntimes()
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate.name); err == nil {
			rt := Runtime{
				Name:     candidate.name,
				Path:     path,
				Priority: candidate.priority,
			}
			rt.Version = rm.getRuntimeVersion(rt.Name)
			rt.Capabilities = rm.detectCapabilities(rt.Name)
			rm.availableRuntimes = append(rm.availableRuntimes, rt)
		}
	}
	sort.Slice(rm.availableRuntimes, func(i, j int) bool {
		return rm.availableRuntimes[i].Priority > rm.availableRuntimes[j].Priority
	})
}

func (rm *Manager) getCandidateRuntimes() []struct {
	name     string
	priority int
} {
	if rm.hardwareProfile.IsAppleSilicon {
		return []struct {
			name     string
			priority int
		}{
			{rtContainer, 100}, // Apple native - FASTEST
			{rtFinch, 80},      // AWS optimized for Mac
			{rtPodman, 60},     // Good alternative
			{rtNerdctl, 50},    // Containerd
			{rtDocker, 30},     // Slowest on Apple Silicon
		}
	}
	return []struct {
		name     string
		priority int
	}{
		{rtPodman, 90},
		{rtDocker, 80},
		{rtNerdctl, 70},
	}
}

// SelectOptimal chooses the best runtime path to use
func (rm *Manager) SelectOptimal() string {
	// Fast path: Apple Silicon prefers Apple Container if present
	if rm.hardwareProfile.IsAppleSilicon {
		for _, rt := range rm.availableRuntimes {
			if rt.Name == rtContainer {
				fmt.Printf("🚀 Using Apple Container (5-10x faster than Docker)\n")
				return rt.Path
			}
		}
	}

	// Determine if we should benchmark now
	if rm.needsBenchmark() {
		fmt.Println("⏱️  Running quick performance test (one-time, ~30s)...")
		rm.benchmarkAll(false)
	}

	best := rm.selectByPerformance()
	if best != "" {
		for _, rt := range rm.availableRuntimes {
			if rt.Name == best {
				rel := rm.getRelativeSpeed(best)
				if rel > 0 {
					fmt.Printf("⚡ Using %s (%.1fx faster)\n", best, rel)
				}
				return rt.Path
			}
		}
	}

	if len(rm.availableRuntimes) > 0 {
		return rm.availableRuntimes[0].Path
	}
	return rtDocker
}

func (rm *Manager) imageAvailable(rt *Runtime, image string) bool {
	if rt == nil {
		return false
	}
	// {rt} images -q image
	cmd := execCommand(rt.Path, "images", "-q", image)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// GetAvailableRuntimes returns all detected runtimes
func (rm *Manager) GetAvailableRuntimes() []Runtime {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.availableRuntimes
}

// GetHardwareProfile returns the detected hardware profile
func (rm *Manager) GetHardwareProfile() HardwareProfile {
	return rm.hardwareProfile
}

// Public helpers used by CLI command handlers
func (rm *Manager) ShowRuntimeInfo() {
	chip := DetectAppleSiliconGeneration()
	fmt.Printf("Hardware: %s (%s/%s)\n", func() string {
		if chip != "" && chip != strUnknown {
			return "Apple " + chip
		}
		return runtime.GOOS
	}(), runtime.GOOS, runtime.GOARCH)

	selected := rm.SelectOptimal()
	selectedName := filepath.Base(selected)
	fmt.Println("Available Runtimes:")
	for _, rt := range rm.availableRuntimes {
		mark := "✅"
		active := ""
		if strings.Contains(selected, rt.Name) || selectedName == rt.Name {
			active = " [ACTIVE]"
		}
		descr := runtimeDescription(rt.Name)
		fmt.Printf("  %s %-10s %-10s%s %s\n", mark, rt.Name, rt.Version, active, descr)
	}

	// Performance section
	if len(rm.benchmarkCache) > 0 {
		// Determine benchmark mode from any entry
		benchMode := "exec-only"
		for _, r := range rm.benchmarkCache {
			if r.Mode == modeBuildExec {
				benchMode = modeBuildExec
				break
			}
		}
		fmt.Println("\nPerformance Scores (relative to fastest):")
		fmt.Printf("Benchmark mode: %s\n", benchMode)
		// stable order by priority
		for _, rt := range rm.availableRuntimes {
			if r, ok := rm.benchmarkCache[rt.Name]; ok && r.Error == "" && r.Score > 0 {
				extra := ""
				if r.Score == 1.0 {
					extra = " (baseline)"
				} else {
					extra = fmt.Sprintf(" %.1fx slower", r.Score)
				}
				fmt.Printf("  %-10s: %.1fx%s\n", rt.Name, r.Score, extra)
			}
		}
	} else {
		fmt.Println("\nNo performance data cached. Run 'mitl runtime benchmark'.")
	}

	// Hints
	hints := OptimizationHints()
	if len(hints) > 0 {
		fmt.Println("\n💡 Optimization Tips:")
		for _, h := range hints {
			fmt.Printf("  • %s\n", h)
		}
	}
}

func runtimeDescription(name string) string {
	switch name {
	case rtContainer:
		return "Native Apple runtime (fastest)"
	case rtFinch:
		return "AWS container runtime"
	case rtDocker:
		return "Docker Desktop"
	case rtPodman:
		return "Podman container engine"
	case rtNerdctl:
		return "containerd CLI"
	default:
		return ""
	}
}

// ForceBenchmark runs benchmarks and displays results
func (rm *Manager) ForceBenchmark(includeBuild bool) {
	rm.benchmarkAll(includeBuild)
	best := rm.selectByPerformance()
	if best != "" {
		rel := rm.getRelativeSpeed(best)
		mode := "exec-only"
		if includeBuild {
			mode = modeBuildExec
		}
		fmt.Printf("Benchmark complete (%s). Best: %s (%.1fx faster)\n", mode, best, rel)
		return
	}
	fmt.Println("Benchmark complete. No successful results; using priority order.")
}

// ShowRecommendations displays optimization recommendations
func (rm *Manager) ShowRecommendations() {
	if rm.hardwareProfile.IsAppleSilicon {
		if _, err := exec.LookPath("container"); err != nil {
			fmt.Println("⚠️  Apple Container not found (would be 5-10x faster)")
			fmt.Println("💡 Install from: developer.apple.com/virtualization")
		}
	}
	_ = rm.SelectOptimal()
	for _, h := range OptimizationHints() {
		fmt.Printf("• %s\n", h)
	}
}

// Simple cross-platform file lock using O_EXCL lock files
func acquireFileLock(lockPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_ = f.Close()
			return nil
		}
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for lock: %s", lockPath)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func releaseFileLock(lockPath string) {
	_ = os.Remove(lockPath)
}
