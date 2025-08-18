package bench

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"mitl/internal/container"
)

// RunBenchmark benchmarks container run operations with different scenarios
type RunBenchmark struct {
	image       string
	command     []string
	interactive bool
	iterations  int
	manager     *container.Manager
}

// NewRunBenchmark creates a new run benchmark with the specified configuration
func NewRunBenchmark(image string, command []string, interactive bool, iterations int) *RunBenchmark {
	return &RunBenchmark{
		image:       image,
		command:     command,
		interactive: interactive,
		iterations:  iterations,
		manager:     container.NewManager(),
	}
}

// NewStartupTimeBenchmark creates a benchmark for container startup time measurement
func NewStartupTimeBenchmark(iterations int) *RunBenchmark {
	return NewRunBenchmark("alpine:latest", []string{"echo", "startup"}, false, iterations)
}

// NewCommandExecutionBenchmark creates a benchmark for command execution time
func NewCommandExecutionBenchmark(iterations int) *RunBenchmark {
	return NewRunBenchmark("alpine:latest",
		[]string{"sh", "-c", "for i in $(seq 1 100); do echo $i; done"}, false, iterations)
}

// NewInteractiveRunBenchmark creates a benchmark for interactive container runs
func NewInteractiveRunBenchmark(iterations int) *RunBenchmark {
	return NewRunBenchmark("alpine:latest", []string{"sh", "-c", "echo 'interactive test'"}, true, iterations)
}

// Setup prepares the benchmark for execution
func (r *RunBenchmark) Setup() error {
	// Ensure we have a container runtime
	runtimes := r.manager.GetAvailableRuntimes()
	if len(runtimes) == 0 {
		return fmt.Errorf("no container runtimes available")
	}

	// Check if the required image is available
	rt := r.manager.SelectOptimal()
	if !r.imageExists(rt, r.image) {
		return fmt.Errorf("required image %s not available, please pull it first", r.image)
	}

	return nil
}

// Run executes one iteration of the run benchmark
func (r *RunBenchmark) Run() (Result, error) {
	result := Result{
		Name:        r.getTestName(),
		Category:    CategoryRun,
		Description: r.getTestDescription(),
		Timestamp:   time.Now(),
		Success:     false,
	}

	// Get memory stats before run
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Generate unique container name
	containerName := fmt.Sprintf("mitl-run-bench-%d", time.Now().UnixNano())

	// Measure startup time, execution time, and cleanup time
	startupTime, execTime, cleanupTime, err := r.executeRun(containerName)
	totalTime := startupTime + execTime + cleanupTime

	// Get memory stats after run
	var memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	if err != nil {
		result.Error = fmt.Sprintf("run failed: %v", err)
		result.TotalTime = Duration{totalTime}
		return result, nil
	}

	// Calculate memory usage
	result.Memory = MemoryStats{
		AllocBytes:      memAfter.Alloc,
		TotalAllocBytes: memAfter.TotalAlloc - memBefore.TotalAlloc,
		SysBytes:        memAfter.Sys,
		NumGC:           memAfter.NumGC - memBefore.NumGC,
		HeapAllocBytes:  memAfter.HeapAlloc,
		HeapSysBytes:    memAfter.HeapSys,
		HeapIdleBytes:   memAfter.HeapIdle,
		HeapInuseBytes:  memAfter.HeapInuse,
	}

	result.TotalTime = Duration{totalTime}
	result.Mean = Duration{totalTime}
	result.Success = true

	// Store detailed timing info in metadata (if needed for analysis)
	// Note: In a real implementation, you might want to add metadata fields to Result
	_ = startupTime // Startup phase time
	_ = execTime    // Execution phase time
	_ = cleanupTime // Cleanup phase time

	return result, nil
}

// executeRun performs the actual container run operation and measures timing
func (r *RunBenchmark) executeRun(containerName string) (startupTime, execTime, cleanupTime time.Duration, err error) {
	rt := r.manager.SelectOptimal()

	// Build the run command
	args := []string{"run", "--name", containerName}

	if r.interactive {
		args = append(args, "-i", "-t")
	}

	// Add remove flag for automatic cleanup (for startup/exec measurement)
	args = append(args, "--rm", r.image)
	args = append(args, r.command...)

	// Measure startup + execution time together
	// (Docker/container runtime combines these phases)
	startTime := time.Now()
	cmd := exec.CommandContext(context.Background(), rt, args...)

	// For interactive tests, we need to handle stdin/stdout
	if r.interactive {
		cmd.Stdin = strings.NewReader("") // Empty input for test
	}

	output, runErr := cmd.CombinedOutput()
	totalRunTime := time.Since(startTime)

	if runErr != nil {
		return 0, 0, 0, fmt.Errorf("run command failed: %w, output: %s", runErr, string(output))
	}

	// For benchmarking purposes, we estimate startup vs execution time
	// In practice, these are hard to separate without container runtime internals
	// We use a heuristic: startup is typically 10-30% of total time for simple commands
	estimatedStartupRatio := 0.2
	startupTime = time.Duration(float64(totalRunTime) * estimatedStartupRatio)
	execTime = totalRunTime - startupTime

	// Cleanup time is minimal since we used --rm, but we measure a separate cleanup operation
	cleanupStart := time.Now()
	r.ensureContainerCleanup(rt, containerName)
	cleanupTime = time.Since(cleanupStart)

	return startupTime, execTime, cleanupTime, nil
}

// ensureContainerCleanup removes the container if it exists
func (r *RunBenchmark) ensureContainerCleanup(rt, containerName string) {
	// Try to remove the container (ignore errors)
	cmd := exec.CommandContext(context.Background(), rt, "rm", "-f", containerName)
	_ = cmd.Run()
}

// imageExists checks if the specified image is available locally
func (r *RunBenchmark) imageExists(rt, image string) bool {
	cmd := exec.CommandContext(context.Background(), rt, "images", "-q", image)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// Cleanup performs post-benchmark cleanup
func (r *RunBenchmark) Cleanup() error {
	// Ensure no leftover containers from failed runs
	rt := r.manager.SelectOptimal()

	// List and remove any containers with our benchmark prefix
	listCmd := exec.CommandContext(context.Background(), rt, "ps", "-a", "-q", "--filter", "name=mitl-run-bench-")
	output, err := listCmd.Output()
	if err != nil {
		return nil // Ignore cleanup errors
	}

	containerIDs := strings.Fields(string(output))
	for _, id := range containerIDs {
		if id != "" {
			rmCmd := exec.CommandContext(context.Background(), rt, "rm", "-f", id)
			_ = rmCmd.Run() // Ignore individual cleanup errors
		}
	}

	return nil
}

// Iterations returns the number of iterations to run
func (r *RunBenchmark) Iterations() int {
	if r.iterations <= 0 {
		return 10 // Default iterations
	}
	return r.iterations
}

// getTestName returns the benchmark name based on configuration
func (r *RunBenchmark) getTestName() string {
	if r.interactive {
		return "run_interactive"
	}

	if len(r.command) > 1 && r.command[0] == "sh" {
		return "run_command_execution"
	}

	return "run_startup_time"
}

// getTestDescription returns a description of what this benchmark tests
func (r *RunBenchmark) getTestDescription() string {
	if r.interactive {
		return "Benchmark interactive container startup and execution time"
	}

	if len(r.command) > 1 && r.command[0] == "sh" {
		return "Benchmark container command execution time"
	}

	return "Benchmark container startup latency"
}
