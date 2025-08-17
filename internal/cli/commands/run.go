package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"mitl/internal/container"
	"mitl/internal/detector"
	"mitl/internal/digest"
	"mitl/internal/volume"
	e "mitl/pkg/errors"
)

// Run executes the given command inside the capsule Docker image.
// This command allows running any command within the project's container environment.
func Run(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: mitl run <command> [args]")
		return fmt.Errorf("no command specified")
	}

	// Use deterministic project digest for capsule tag
	digestValue, derr := digest.ProjectTag(".", digest.Options{Algorithm: "sha256"})
	if derr != nil {
		return e.Wrap(derr, e.ErrUnknown, "Failed to compute project digest").
			WithSuggestion("Run 'mitl digest --verbose' for details")
	}
	tag := fmt.Sprintf("mitl-capsule:%s", digestValue)

	// Detect project type for proper volume mounting and pnpm enforcement
	detectorInstance := detector.NewProjectDetector("")
	_ = detectorInstance.Detect()

	// Initialize volume manager
	cli := findRunCLI()
	vm := volume.NewManager(cli, "")

	// Intercept: npm/yarn converted to pnpm for Node projects
	if detectorInstance.Type == detector.TypeNodeGeneric || strings.HasPrefix(string(detectorInstance.Type), "node") {
		args = vm.InterceptNodeCommand(args)
		pnpm := volume.NewPnpmManager("", vm)
		_ = pnpm.ConvertToUsingPnpm()
	}

	// Build container args with mounts
	containerArgs := []string{"run", "--rm"}
	containerArgs = append(containerArgs, vm.GetMounts(detectorInstance.Type)...)
	// If performing package installs in Node containers, run as root to avoid permission issues on mounted volumes
	joined := strings.Join(args, " ")
	if strings.HasPrefix(string(detectorInstance.Type), "node") {
		if strings.Contains(joined, "install") || strings.Contains(joined, "add") || strings.Contains(joined, "ci") {
			containerArgs = append(containerArgs, "--user", "0")
		}
	}
	containerArgs = append(containerArgs, "-w", "/app")
	containerArgs = append(containerArgs, tag)
	containerArgs = append(containerArgs, args...)

	cmd := execCommand(cli, containerArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		// Map common runtime issues to MitlError with guidance
		msg := strings.ToLower(err.Error())
		// docker daemon not running / connection issues
		if strings.Contains(msg, "docker daemon is not running") ||
			strings.Contains(msg, "cannot connect to the docker daemon") ||
			strings.Contains(msg, "error during connect") ||
			strings.Contains(msg, "dial unix") ||
			strings.Contains(msg, "connect: connection refused") {
			return e.New(e.ErrRuntimeNotRunning, "Container runtime is not running").WithContext("runtime", cli)
		}
		// permission denied on socket
		if strings.Contains(msg, "permission denied") {
			return e.New(e.ErrRuntimePermission, "Permission denied when accessing runtime").WithContext("runtime", cli).WithCause(err)
		}
		// runtime binary missing
		if strings.Contains(msg, "executable file not found") || strings.Contains(msg, "file not found") {
			return e.New(e.ErrRuntimeNotFound, "No container runtime found").WithContext("runtime", cli).WithCause(err)
		}
		return e.Wrap(err, e.ErrUnknown, "Failed to run command").WithContext("runtime", cli)
	}
	return nil
}

// findRunCLI attempts to locate a suitable container run CLI. The logic
// mirrors findBuildCLI but allows override via MITL_RUN_CLI. In practice,
// the same binary can be used for building and running, but having two
// separate functions allows for future differences in behaviour if needed.
func findRunCLI() string {
	// Environment variable takes highest priority
	if env := os.Getenv("MITL_RUN_CLI"); env != "" {
		if _, err := exec.LookPath(env); err == nil {
			return env
		}
	}
	// Check configuration file for user preference
	cfg := loadConfig()
	if cfg.RunCLI != "" {
		if _, err := exec.LookPath(cfg.RunCLI); err == nil {
			return cfg.RunCLI
		}
	}
	// Intelligent selection (use same runtime for consistency)
	rm := container.NewManager()
	return rm.SelectOptimal()
}
