package errors

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

const defaultRuntime = "docker"

// RecoveryStrategy defines how to recover from an error
type RecoveryStrategy interface {
	CanRecover(err *MitlError) bool
	Attempt(err *MitlError) error
	Description() string
}

// Recoverer attempts to recover from errors
type Recoverer struct {
	strategies []RecoveryStrategy
	maxRetries int
	verbose    bool
}

// NewRecoverer creates a new error recoverer
func NewRecoverer(verbose bool) *Recoverer {
	return &Recoverer{
		strategies: []RecoveryStrategy{
			&RuntimeStartStrategy{},
			&CacheClearStrategy{},
			&NetworkRetryStrategy{},
			&DiskSpaceStrategy{},
		},
		maxRetries: 3,
		verbose:    verbose,
	}
}

// Recover attempts to recover from an error
func (r *Recoverer) Recover(err *MitlError) error {
	if !err.Recoverable {
		return err
	}
	for _, strategy := range r.strategies {
		if strategy.CanRecover(err) {
			if r.verbose {
				fmt.Printf("🔧 Attempting recovery: %s\n", strategy.Description())
			}
			if recErr := strategy.Attempt(err); recErr == nil {
				fmt.Println("✅ Recovery successful!")
				return nil
			} else if r.verbose {
				fmt.Printf("⚠️  Recovery failed: %v\n", recErr)
			}
		}
	}
	return err
}

// RuntimeStartStrategy tries to start the container runtime
type RuntimeStartStrategy struct{}

func (s *RuntimeStartStrategy) CanRecover(err *MitlError) bool {
	return err.Code == ErrRuntimeNotRunning
}

func (s *RuntimeStartStrategy) Attempt(err *MitlError) error {
	runtime := err.Context["runtime"]
	if runtime == "" {
		runtime = defaultRuntime
	}

	fmt.Printf("🚀 Starting %s...\n", runtime)
	var cmd *exec.Cmd
	switch runtime {
	case defaultRuntime:
		if _, statErr := os.Stat("/Applications/Docker.app"); statErr == nil {
			cmd = exec.Command("open", "-a", "Docker")
		} else {
			cmd = exec.Command("systemctl", "start", "docker")
		}
	case "podman":
		cmd = exec.Command("podman", "machine", "start")
	default:
		return fmt.Errorf("unknown runtime: %s", runtime)
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Println("⏳ Waiting for runtime to be ready...")
	for i := 0; i < 30; i++ {
		checkCmd := exec.Command(runtime, "version")
		if err := checkCmd.Run(); err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("%s failed to start", runtime)
}

func (s *RuntimeStartStrategy) Description() string { return "Starting container runtime" }

// CacheClearStrategy clears a corrupted cache
type CacheClearStrategy struct{}

func (s *CacheClearStrategy) CanRecover(err *MitlError) bool {
	return err.Code == ErrCacheCorrupted
}

func (s *CacheClearStrategy) Attempt(err *MitlError) error {
	fmt.Println("🧹 Clearing corrupted cache...")
	cacheDir := os.ExpandEnv("$HOME/.mitl/cache")
	if rmErr := os.RemoveAll(cacheDir); rmErr != nil {
		return fmt.Errorf("failed to clear cache: %w", rmErr)
	}
	if mkErr := os.MkdirAll(cacheDir, 0o755); mkErr != nil {
		return fmt.Errorf("failed to recreate cache: %w", mkErr)
	}
	return nil
}

func (s *CacheClearStrategy) Description() string { return "Clearing corrupted cache" }

// NetworkRetryStrategy triggers a retry for network operations
type NetworkRetryStrategy struct{}

func (s *NetworkRetryStrategy) CanRecover(err *MitlError) bool {
	return err.Code == ErrNetworkTimeout || err.Code == ErrRegistryUnreachable
}

func (s *NetworkRetryStrategy) Attempt(err *MitlError) error {
	fmt.Println("🔄 Retrying network operation...")
	time.Sleep(2 * time.Second)
	return nil
}

func (s *NetworkRetryStrategy) Description() string { return "Retrying network operation" }

// DiskSpaceStrategy frees disk space using the runtime
type DiskSpaceStrategy struct{}

func (s *DiskSpaceStrategy) CanRecover(err *MitlError) bool { return err.Code == ErrDiskFull }

func (s *DiskSpaceStrategy) Attempt(err *MitlError) error {
	fmt.Println("💾 Attempting to free disk space...")
	runtime := err.Context["runtime"]
	if runtime == "" {
		runtime = defaultRuntime
	}
	cmd := exec.Command(runtime, "image", "prune", "-f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to prune images: %w", err)
	}
	cmd = exec.Command(runtime, "builder", "prune", "-f")
	_ = cmd.Run() // not all runtimes support this
	fmt.Println("✅ Freed disk space")
	return nil
}

func (s *DiskSpaceStrategy) Description() string { return "Freeing disk space" }
