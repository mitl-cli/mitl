package bench

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const naString = "N/A"

// BenchmarkConfig represents the configuration needed to run a benchmark with container tools
type BenchmarkConfig struct {
	Image        string            `json:"image"`
	Command      string            `json:"command"`
	VolumeMounts []string          `json:"volume_mounts"`
	Environment  map[string]string `json:"environment"`
}

// Comparison holds the results of a comparison between mitl and other container tools
type Comparison struct {
	Tool    string   // "docker" or "podman"
	Results []Result // Results from the comparison tool
}

// ComparisonReport contains the results from mitl and comparison tools
type ComparisonReport struct {
	mitlResults   []Result
	dockerResults []Result
	podmanResults []Result
}

// NewComparisonReport creates a new comparison report with mitl results
func NewComparisonReport(mitlResults []Result) *ComparisonReport {
	return &ComparisonReport{
		mitlResults: mitlResults,
	}
}

// RunDockerComparison runs the same benchmarks using Docker and returns the results
func RunDockerComparison(benchmarks []Benchmark) ([]Result, error) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker not found in PATH: %w", err)
	}

	var results []Result
	for _, bench := range benchmarks {
		// Extract configuration from benchmark runner if it implements a config interface
		config := extractBenchmarkConfig(bench)
		result := runDockerEquivalent(bench, config)
		results = append(results, result)
	}

	return results, nil
}

// RunPodmanComparison runs the same benchmarks using Podman and returns the results
func RunPodmanComparison(benchmarks []Benchmark) ([]Result, error) {
	// Check if Podman is available
	if _, err := exec.LookPath("podman"); err != nil {
		return nil, fmt.Errorf("podman not found in PATH: %w", err)
	}

	var results []Result
	for _, bench := range benchmarks {
		// Extract configuration from benchmark runner if it implements a config interface
		config := extractBenchmarkConfig(bench)
		result := runPodmanEquivalent(bench, config)
		results = append(results, result)
	}

	return results, nil
}

// extractBenchmarkConfig attempts to extract configuration from a benchmark
// This is a placeholder implementation - in real usage, you'd implement this based on your specific benchmark runner types
func extractBenchmarkConfig(bench Benchmark) BenchmarkConfig {
	// Default configuration for demonstration
	// In a real implementation, you'd extract this from the benchmark runner
	return BenchmarkConfig{
		Image:        "alpine:latest",
		Command:      "echo 'benchmark test'",
		VolumeMounts: []string{},
		Environment:  map[string]string{},
	}
}

// runDockerEquivalent runs a single benchmark using Docker
func runDockerEquivalent(bench Benchmark, config BenchmarkConfig) Result {
	start := time.Now()

	// Build the Docker command equivalent
	args := []string{"run", "--rm"}

	// Add volume mounts if specified
	for _, mount := range config.VolumeMounts {
		args = append(args, "-v", mount)
	}

	// Add environment variables if specified
	for key, value := range config.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add the image and command
	args = append(args, config.Image)
	if config.Command != "" {
		// Split command into arguments
		cmdParts := strings.Fields(config.Command)
		args = append(args, cmdParts...)
	}

	// Execute the Docker command
	cmd := exec.Command("docker", args...)
	err := cmd.Run()
	duration := time.Since(start)

	result := Result{
		Name:        bench.Name + "_docker",
		Category:    bench.Category,
		Description: bench.Description + " (Docker)",
		Iterations:  1, // Single run for comparison
		Success:     err == nil,
		Timestamp:   start,
	}

	if err != nil {
		result.Error = err.Error()
	}

	// Set durations - using the custom Duration type
	result.Mean = Duration{duration}
	result.Median = Duration{duration}
	result.Min = Duration{duration}
	result.Max = Duration{duration}
	result.StdDev = Duration{0} // Single run, no deviation
	result.P95 = Duration{duration}
	result.P99 = Duration{duration}
	result.TotalTime = Duration{duration}

	return result
}

// runPodmanEquivalent runs a single benchmark using Podman
func runPodmanEquivalent(bench Benchmark, config BenchmarkConfig) Result {
	start := time.Now()

	// Build the Podman command equivalent
	args := []string{"run", "--rm"}

	// Add volume mounts if specified
	for _, mount := range config.VolumeMounts {
		args = append(args, "-v", mount)
	}

	// Add environment variables if specified
	for key, value := range config.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add the image and command
	args = append(args, config.Image)
	if config.Command != "" {
		// Split command into arguments
		cmdParts := strings.Fields(config.Command)
		args = append(args, cmdParts...)
	}

	// Execute the Podman command
	cmd := exec.Command("podman", args...)
	err := cmd.Run()
	duration := time.Since(start)

	result := Result{
		Name:        bench.Name + "_podman",
		Category:    bench.Category,
		Description: bench.Description + " (Podman)",
		Iterations:  1, // Single run for comparison
		Success:     err == nil,
		Timestamp:   start,
	}

	if err != nil {
		result.Error = err.Error()
	}

	// Set durations - using the custom Duration type
	result.Mean = Duration{duration}
	result.Median = Duration{duration}
	result.Min = Duration{duration}
	result.Max = Duration{duration}
	result.StdDev = Duration{0} // Single run, no deviation
	result.P95 = Duration{duration}
	result.P99 = Duration{duration}
	result.TotalTime = Duration{duration}

	return result
}

// AddDockerResults adds Docker comparison results to the report
func (cr *ComparisonReport) AddDockerResults(results []Result) {
	cr.dockerResults = results
}

// AddPodmanResults adds Podman comparison results to the report
func (cr *ComparisonReport) AddPodmanResults(results []Result) {
	cr.podmanResults = results
}

// Generate creates a formatted comparison table showing mitl vs Docker vs Podman performance
func (cr *ComparisonReport) Generate() string {
	var report strings.Builder

	report.WriteString("\n=== Performance Comparison Report ===\n\n")

	// Table header
	report.WriteString(fmt.Sprintf("%-30s %-15s %-15s %-15s %-12s %-12s\n",
		"Benchmark", "mitl (ms)", "Docker (ms)", "Podman (ms)", "vs Docker", "vs Podman"))
	report.WriteString(strings.Repeat("-", 110) + "\n")

	// Find matching benchmarks by name prefix (remove tool suffix)
	benchmarkMap := make(map[string]map[string]Result)

	// Group mitl results
	for _, result := range cr.mitlResults {
		if benchmarkMap[result.Name] == nil {
			benchmarkMap[result.Name] = make(map[string]Result)
		}
		benchmarkMap[result.Name]["mitl"] = result
	}

	// Group Docker results
	for _, result := range cr.dockerResults {
		benchName := strings.TrimSuffix(result.Name, "_docker")
		if benchmarkMap[benchName] == nil {
			benchmarkMap[benchName] = make(map[string]Result)
		}
		benchmarkMap[benchName]["docker"] = result
	}

	// Group Podman results
	for _, result := range cr.podmanResults {
		benchName := strings.TrimSuffix(result.Name, "_podman")
		if benchmarkMap[benchName] == nil {
			benchmarkMap[benchName] = make(map[string]Result)
		}
		benchmarkMap[benchName]["podman"] = result
	}

	// Generate comparison rows
	for benchName, results := range benchmarkMap {
		mitlResult, hasMitl := results["mitl"]
		dockerResult, hasDocker := results["docker"]
		podmanResult, hasPodman := results["podman"]

    mitlTime := naString
    dockerTime := naString
    podmanTime := naString
    dockerSpeedup := naString
    podmanSpeedup := naString

		if hasMitl {
			mitlTime = fmt.Sprintf("%.2f", float64(mitlResult.Mean.Nanoseconds())/1e6)
		}

		if hasDocker {
			dockerTime = fmt.Sprintf("%.2f", float64(dockerResult.Mean.Nanoseconds())/1e6)
			if hasMitl && mitlResult.Mean.Duration > 0 {
				speedup := calculateSpeedup(
					float64(mitlResult.Mean.Nanoseconds())/1e6,
					float64(dockerResult.Mean.Nanoseconds())/1e6,
				)
				dockerSpeedup = formatSpeedup(speedup)
			}
		}

		if hasPodman {
			podmanTime = fmt.Sprintf("%.2f", float64(podmanResult.Mean.Nanoseconds())/1e6)
			if hasMitl && mitlResult.Mean.Duration > 0 {
				speedup := calculateSpeedup(
					float64(mitlResult.Mean.Nanoseconds())/1e6,
					float64(podmanResult.Mean.Nanoseconds())/1e6,
				)
				podmanSpeedup = formatSpeedup(speedup)
			}
		}

		report.WriteString(fmt.Sprintf("%-30s %-15s %-15s %-15s %-12s %-12s\n",
			benchName, mitlTime, dockerTime, podmanTime, dockerSpeedup, podmanSpeedup))
	}

	report.WriteString("\n")
	report.WriteString("Notes:\n")
	report.WriteString("- Speedup > 1.0x means mitl is faster\n")
	report.WriteString("- Speedup < 1.0x means comparison tool is faster\n")
	report.WriteString("- Times are in milliseconds (mean duration)\n")

	return report.String()
}

// calculateSpeedup calculates the speedup ratio (comparison_time / mitl_time)
// A ratio > 1.0 means mitl is faster, < 1.0 means comparison tool is faster
func calculateSpeedup(mitlTime, comparisonTime float64) float64 {
	if mitlTime <= 0 {
		return 0
	}
	return comparisonTime / mitlTime
}

// formatSpeedup formats a speedup ratio for display
func formatSpeedup(speedup float64) string {
	if speedup <= 0 {
		return naString
	}
	return fmt.Sprintf("%.2fx", speedup)
}

// Statistics represents statistical measurements for a result
// This is a helper struct to match the expected format from the task description
type Statistics struct {
	Mean   float64 `json:"mean"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	StdDev float64 `json:"std_dev"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
}

// ToStatistics converts the existing Result durations to a Statistics struct for validation
func (r *Result) ToStatistics() Statistics {
	return Statistics{
		Mean:   float64(r.Mean.Nanoseconds()) / 1e6,
		Min:    float64(r.Min.Nanoseconds()) / 1e6,
		Max:    float64(r.Max.Nanoseconds()) / 1e6,
		StdDev: float64(r.StdDev.Nanoseconds()) / 1e6,
		P95:    float64(r.P95.Nanoseconds()) / 1e6,
		P99:    float64(r.P99.Nanoseconds()) / 1e6,
	}
}
