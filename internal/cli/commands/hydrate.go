package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"mitl/internal/build"
	"mitl/internal/cache"
	"mitl/internal/container"
	"mitl/internal/detector"
	"mitl/internal/digest"

	e "mitl/pkg/errors"
)

// testable indirections for filesystem and time
var (
	mkTempDir = os.MkdirTemp
	writeFile = os.WriteFile
	timeNowFn = time.Now
)

// execCommand enables test stubbing for command execution
var execCommand = exec.Command

// Config holds user preferences for container runtimes. BuildCLI and RunCLI
// store the chosen CLI commands for building and running capsules. These
// settings are saved in a JSON file at ~/.mitl.json.
type Config struct {
	BuildCLI string `json:"build_cli"`
	RunCLI   string `json:"run_cli"`
	// LastBuildSeconds stores the duration in seconds of the last successful
	// build for a given digest key. Used to show time saved on cache hits.
	LastBuildSeconds map[string]float64 `json:"last_build_seconds,omitempty"`
}

// Hydrate builds a Docker image for the current project using a temporary Dockerfile.
// This command creates an optimized container image (capsule) for the detected project type.
func Hydrate(args []string) error {
	start := time.Now()
	// Use deterministic project digest for capsule tag
	digestValue, derr := digest.ProjectTag(".", digest.Options{Algorithm: "sha256"})
	if derr != nil {
		return e.Wrap(derr, e.ErrUnknown, "Failed to compute project digest").
			WithSuggestion("Run 'mitl digest --verbose' for details")
	}
	tag := fmt.Sprintf("mitl-capsule:%s", digestValue)

	buildCmd := findBuildCLI()
	cache := cache.NewCapsuleCache(buildCmd, tag)
	exists, err := cache.Exists()
	if err != nil {
		fmt.Printf("\x1b[33m⚠️  Cache check failed: %v\x1b[0m\n", err)
	} else if exists && cache.ValidateDigest(digestValue) {
		elapsed := time.Since(start)
		cfg := loadConfig()
		saved := 0.0
		if cfg.LastBuildSeconds != nil {
			if v, ok := cfg.LastBuildSeconds[digestValue]; ok {
				saved = v
			}
		}
		if saved > 0 {
			fmt.Printf("\x1b[32m✨ Using cached capsule: %s (%.2fs, saved %.1fs)\x1b[0m\n", tag, elapsed.Seconds(), saved)
		} else {
			fmt.Printf("\x1b[32m✨ Using cached capsule: %s (%.2fs)\x1b[0m\n", tag, elapsed.Seconds())
		}
		return nil
	}

	fmt.Printf("\x1b[33m🔍 Analyzing project structure...\x1b[0m\n")
	detectorInstance := detector.NewProjectDetector("")
	if derr := detectorInstance.Detect(); derr != nil {
		fmt.Printf("\x1b[33m⚠️  Detection failed: %v — using generic Dockerfile\x1b[0m\n", derr)
	}
	generator := NewDockerfileGenerator(detectorInstance)
	dockerfileContent, gerr := generator.Generate()
	if gerr != nil {
		fmt.Printf("\x1b[31m❌ Dockerfile generation failed: %v\x1b[0m\n", gerr)
		return gerr
	}
	if detectorInstance.Type != detector.TypeUnknown {
		fmt.Printf("\x1b[32m📦 Detected: %s\x1b[0m\n", detectorInstance.Type)
		if detectorInstance.Framework != "" {
			fmt.Printf("\x1b[32m🚀 Framework: %s %s\x1b[0m\n", detectorInstance.Framework, detectorInstance.Version)
		}
	}
	for _, hint := range generator.OptimizationHints() {
		fmt.Println(hint)
	}

	fmt.Printf("\x1b[33m🔨 Building optimized capsule: %s\x1b[0m\n", tag)
	tmpDir, err := mkTempDir("", "mitl-build-")
	if err != nil {
		return e.Wrap(err, e.ErrPermissionDenied, "Failed to create temp directory")
	}
	defer os.RemoveAll(tmpDir)
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if werr := writeFile(dockerfilePath, []byte(dockerfileContent), 0o644); werr != nil {
		return e.Wrap(werr, e.ErrPermissionDenied, "Failed to write Dockerfile")
	}
	// Determine the target platform. BuildKit can autoselect, but we set explicitly when helpful.
	platform := resolveBuildPlatform()
	args = []string{"build", "-t", tag}
	if platform != "" {
		args = append(args, "--platform", platform)
	}
	args = append(args, "-f", dockerfilePath, ".")
	cmd := execCommand(buildCmd, args...)
	// Stream output while also capturing stderr to detect disk-full conditions
	var errBuf bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &errBuf)
	buildStart := timeNowFn()
	err = cmd.Run()
	if err != nil {
		// Basic disk space diagnostic
		lowerOut := strings.ToLower(errBuf.String() + "\n" + err.Error())
		if strings.Contains(lowerOut, "no space left on device") || strings.Contains(lowerOut, "no space left") {
			derr := e.New(e.ErrDiskFull, "Not enough disk space").WithCause(err).WithContext("runtime", buildCmd)
			return derr
		}
		return e.Wrap(err, e.ErrBuildFailed, "Build failed").WithContext("runtime", buildCmd)
	}
	buildElapsed := time.Since(buildStart)
	fmt.Printf("\x1b[32mCapsule built: %s (%.1fs)\x1b[0m\n", tag, buildElapsed.Seconds())

	// Persist build duration for future "time saved" messaging
	cfg := loadConfig()
	if cfg.LastBuildSeconds == nil {
		cfg.LastBuildSeconds = make(map[string]float64)
	}
	cfg.LastBuildSeconds[digestValue] = buildElapsed.Seconds()
	saveConfig(cfg)
	return nil
}

// configPath returns the absolute path to the mitl configuration file. It
// uses the HOME environment variable if present, otherwise falls back to the
// current working directory. The file is named .mitl.json.
func configPath() string {
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.Getwd()
	}
	return filepath.Join(home, ".mitl.json")
}

// loadConfig reads the configuration from the config file. If the file
// doesn't exist or cannot be parsed, a zero-value Config is returned.
func loadConfig() Config {
	var cfg Config
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

// saveConfig writes the provided configuration to the config file. Errors
// are silently ignored because configuration is optional.
func saveConfig(cfg Config) {
	path := configPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

// resolveBuildPlatform returns the platform flag based on env/arch.
func resolveBuildPlatform() string {
	if p := os.Getenv("MITL_PLATFORM"); p != "" {
		return p
	}
	if runtime.GOARCH == "arm64" {
		return "linux/arm64"
	}
	return ""
}

// findBuildCLI attempts to locate a suitable container build CLI. The
// priority order is influenced by the host OS. On macOS, the native
// `container` CLI (from Apple's Containerization framework) is preferred
// when present, followed by finch, podman and nerdctl. On other OSes,
// podman and nerdctl are preferred. Docker is always used as the last
// resort. Environment variables and user configuration override the
// auto‑detection logic.
func findBuildCLI() string {
	// Environment variable takes highest priority
	if env := os.Getenv("MITL_BUILD_CLI"); env != "" {
		if _, err := exec.LookPath(env); err == nil {
			return env
		}
	}
	// Check configuration file for user preference
	cfg := loadConfig()
	if cfg.BuildCLI != "" {
		if _, err := exec.LookPath(cfg.BuildCLI); err == nil {
			return cfg.BuildCLI
		}
	}
	// Intelligent selection
	rm := container.NewManager()
	return rm.SelectOptimal()
}

// NewDockerfileGenerator creates a legacy wrapper for backwards compatibility
func NewDockerfileGenerator(detector *detector.ProjectDetector) *build.LegacyDockerfileGenerator {
	return build.NewLegacyDockerfileGenerator(detector)
}
