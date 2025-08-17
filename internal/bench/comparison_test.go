package bench

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestRunDockerComparison tests Docker comparison functionality
func TestRunDockerComparison(t *testing.T) {
	tests := []struct {
		name        string
		benchmarks  []Benchmark
		mockDocker  bool
		expectError bool
	}{
		{
			name:        "empty benchmarks",
			benchmarks:  []Benchmark{},
			mockDocker:  true,
			expectError: false,
		},
		{
			name: "single benchmark with docker available",
			benchmarks: []Benchmark{
				{
					Name:        "test_benchmark",
					Description: "Test benchmark",
					Category:    CategoryRun,
				},
			},
			mockDocker:  true,
			expectError: false,
		},
		{
			name: "multiple benchmarks",
			benchmarks: []Benchmark{
				{
					Name:        "benchmark1",
					Description: "First benchmark",
					Category:    CategoryRun,
				},
				{
					Name:        "benchmark2",
					Description: "Second benchmark",
					Category:    CategoryBuild,
				},
			},
			mockDocker:  true,
			expectError: false,
		},
		{
			name: "docker not available",
			benchmarks: []Benchmark{
				{
					Name:        "test_benchmark",
					Description: "Test benchmark",
					Category:    CategoryRun,
				},
			},
			mockDocker:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock docker availability
			originalPath := os.Getenv("PATH")
			if tt.mockDocker {
				// Create a temporary directory with a mock docker command
				mockPath := createMockDocker(t)
				defer os.RemoveAll(mockPath)
				os.Setenv("PATH", mockPath+":"+originalPath)
			} else {
				// Set empty PATH to simulate docker not being available
				os.Setenv("PATH", "")
			}
			defer os.Setenv("PATH", originalPath)

			results, err := RunDockerComparison(tt.benchmarks)

			if (err != nil) != tt.expectError {
				t.Errorf("RunDockerComparison() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if len(results) != len(tt.benchmarks) {
					t.Errorf("Expected %d results, got %d", len(tt.benchmarks), len(results))
				}

				// Verify each result has the correct naming and structure
				for i, result := range results {
					expectedName := tt.benchmarks[i].Name + "_docker"
					if result.Name != expectedName {
						t.Errorf("Result[%d] name = %v, want %v", i, result.Name, expectedName)
					}

					if result.Category != tt.benchmarks[i].Category {
						t.Errorf("Result[%d] category = %v, want %v", i, result.Category, tt.benchmarks[i].Category)
					}

					expectedDesc := tt.benchmarks[i].Description + " (Docker)"
					if result.Description != expectedDesc {
						t.Errorf("Result[%d] description = %v, want %v", i, result.Description, expectedDesc)
					}

					// Verify timing information is present
					if result.TotalTime.Duration <= 0 {
						t.Errorf("Result[%d] should have positive total time", i)
					}
				}
			}
		})
	}
}

// TestRunPodmanComparison tests Podman comparison functionality
func TestRunPodmanComparison(t *testing.T) {
	tests := []struct {
		name        string
		benchmarks  []Benchmark
		mockPodman  bool
		expectError bool
	}{
		{
			name:        "empty benchmarks",
			benchmarks:  []Benchmark{},
			mockPodman:  true,
			expectError: false,
		},
		{
			name: "single benchmark with podman available",
			benchmarks: []Benchmark{
				{
					Name:        "test_benchmark",
					Description: "Test benchmark",
					Category:    CategoryRun,
				},
			},
			mockPodman:  true,
			expectError: false,
		},
		{
			name: "podman not available",
			benchmarks: []Benchmark{
				{
					Name:        "test_benchmark",
					Description: "Test benchmark",
					Category:    CategoryRun,
				},
			},
			mockPodman:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock podman availability
			originalPath := os.Getenv("PATH")
			if tt.mockPodman {
				// Create a temporary directory with a mock podman command
				mockPath := createMockPodman(t)
				defer os.RemoveAll(mockPath)
				os.Setenv("PATH", mockPath+":"+originalPath)
			} else {
				// Set empty PATH to simulate podman not being available
				os.Setenv("PATH", "")
			}
			defer os.Setenv("PATH", originalPath)

			results, err := RunPodmanComparison(tt.benchmarks)

			if (err != nil) != tt.expectError {
				t.Errorf("RunPodmanComparison() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if len(results) != len(tt.benchmarks) {
					t.Errorf("Expected %d results, got %d", len(tt.benchmarks), len(results))
				}

				// Verify each result has the correct naming and structure
				for i, result := range results {
					expectedName := tt.benchmarks[i].Name + "_podman"
					if result.Name != expectedName {
						t.Errorf("Result[%d] name = %v, want %v", i, result.Name, expectedName)
					}

					if result.Category != tt.benchmarks[i].Category {
						t.Errorf("Result[%d] category = %v, want %v", i, result.Category, tt.benchmarks[i].Category)
					}

					expectedDesc := tt.benchmarks[i].Description + " (Podman)"
					if result.Description != expectedDesc {
						t.Errorf("Result[%d] description = %v, want %v", i, result.Description, expectedDesc)
					}
				}
			}
		})
	}
}

// TestSpeedupCalculations tests speedup calculations
func TestSpeedupCalculations(t *testing.T) {
	tests := []struct {
		name            string
		mitlTime        float64
		comparisonTime  float64
		expectedSpeedup float64
	}{
		{
			name:            "mitl faster",
			mitlTime:        100.0,
			comparisonTime:  200.0,
			expectedSpeedup: 2.0,
		},
		{
			name:            "comparison faster",
			mitlTime:        200.0,
			comparisonTime:  100.0,
			expectedSpeedup: 0.5,
		},
		{
			name:            "equal performance",
			mitlTime:        100.0,
			comparisonTime:  100.0,
			expectedSpeedup: 1.0,
		},
		{
			name:            "zero mitl time",
			mitlTime:        0.0,
			comparisonTime:  100.0,
			expectedSpeedup: 0.0,
		},
		{
			name:            "very small mitl time",
			mitlTime:        0.001,
			comparisonTime:  100.0,
			expectedSpeedup: 100000.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSpeedup(tt.mitlTime, tt.comparisonTime)
			if result != tt.expectedSpeedup {
				t.Errorf("calculateSpeedup() = %v, want %v", result, tt.expectedSpeedup)
			}
		})
	}
}

// TestFormatSpeedup tests speedup formatting
func TestFormatSpeedup(t *testing.T) {
	tests := []struct {
		name     string
		speedup  float64
		expected string
	}{
		{
			name:     "normal speedup",
			speedup:  2.5,
			expected: "2.50x",
		},
		{
			name:     "very high speedup",
			speedup:  100.0,
			expected: "100.00x",
		},
		{
			name:     "fractional speedup",
			speedup:  0.75,
			expected: "0.75x",
		},
		{
			name:     "zero speedup",
			speedup:  0.0,
			expected: "N/A",
		},
		{
			name:     "negative speedup",
			speedup:  -1.0,
			expected: "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSpeedup(tt.speedup)
			if result != tt.expected {
				t.Errorf("formatSpeedup() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestComparisonReport tests comparison report generation
func TestComparisonReport(t *testing.T) {
	// Create test results
	mitlResults := []Result{
		{
			Name:        "benchmark1",
			Description: "First benchmark",
			Category:    CategoryRun,
			Mean:        Duration{100 * time.Millisecond},
			Success:     true,
		},
		{
			Name:        "benchmark2",
			Description: "Second benchmark",
			Category:    CategoryBuild,
			Mean:        Duration{200 * time.Millisecond},
			Success:     true,
		},
	}

	dockerResults := []Result{
		{
			Name:        "benchmark1_docker",
			Description: "First benchmark (Docker)",
			Category:    CategoryRun,
			Mean:        Duration{150 * time.Millisecond},
			Success:     true,
		},
		{
			Name:        "benchmark2_docker",
			Description: "Second benchmark (Docker)",
			Category:    CategoryBuild,
			Mean:        Duration{250 * time.Millisecond},
			Success:     true,
		},
	}

	podmanResults := []Result{
		{
			Name:        "benchmark1_podman",
			Description: "First benchmark (Podman)",
			Category:    CategoryRun,
			Mean:        Duration{120 * time.Millisecond},
			Success:     true,
		},
		{
			Name:        "benchmark2_podman",
			Description: "Second benchmark (Podman)",
			Category:    CategoryBuild,
			Mean:        Duration{230 * time.Millisecond},
			Success:     true,
		},
	}

	// Create comparison report
	report := NewComparisonReport(mitlResults)
	report.AddDockerResults(dockerResults)
	report.AddPodmanResults(podmanResults)

	// Generate the report
	output := report.Generate()

	// Verify report contains expected elements
	if !strings.Contains(output, "Performance Comparison Report") {
		t.Error("Report should contain title")
	}

	if !strings.Contains(output, "benchmark1") {
		t.Error("Report should contain benchmark1")
	}

	if !strings.Contains(output, "benchmark2") {
		t.Error("Report should contain benchmark2")
	}

	if !strings.Contains(output, "vs Docker") {
		t.Error("Report should contain Docker comparison column")
	}

	if !strings.Contains(output, "vs Podman") {
		t.Error("Report should contain Podman comparison column")
	}

	// Verify speedup calculations are present
	// mitl (100ms) vs docker (150ms) = 1.5x speedup for mitl
	if !strings.Contains(output, "1.50x") {
		t.Error("Report should contain speedup calculations")
	}

	// Verify notes section
	if !strings.Contains(output, "Notes:") {
		t.Error("Report should contain notes section")
	}

	if !strings.Contains(output, "Times are in milliseconds") {
		t.Error("Report should contain timing units explanation")
	}
}

// TestExtractBenchmarkConfig tests benchmark configuration extraction
func TestExtractBenchmarkConfig(t *testing.T) {
	benchmark := Benchmark{
		Name:        "test_benchmark",
		Description: "Test benchmark",
		Category:    CategoryRun,
	}

	config := extractBenchmarkConfig(benchmark)

	// Verify default configuration
	if config.Image != "alpine:latest" {
		t.Errorf("Default image = %v, want alpine:latest", config.Image)
	}

	if config.Command != "echo 'benchmark test'" {
		t.Errorf("Default command = %v, want echo 'benchmark test'", config.Command)
	}

	if len(config.VolumeMounts) != 0 {
		t.Errorf("Default volume mounts should be empty, got %v", config.VolumeMounts)
	}

	if len(config.Environment) != 0 {
		t.Errorf("Default environment should be empty, got %v", config.Environment)
	}
}

// TestRunDockerEquivalent tests Docker equivalent execution with mocking
func TestRunDockerEquivalent(t *testing.T) {
	// Create a temporary directory with a mock docker command
	mockPath := createMockDocker(t)
	defer os.RemoveAll(mockPath)

	// Set PATH to use mock docker
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", mockPath+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	benchmark := Benchmark{
		Name:        "test_benchmark",
		Description: "Test benchmark",
		Category:    CategoryRun,
	}

	config := BenchmarkConfig{
		Image:        "alpine:latest",
		Command:      "echo 'test'",
		VolumeMounts: []string{"/tmp:/tmp"},
		Environment:  map[string]string{"TEST_VAR": "test_value"},
	}

	result := runDockerEquivalent(benchmark, config)

	// Verify result structure
	if result.Name != "test_benchmark_docker" {
		t.Errorf("Result name = %v, want test_benchmark_docker", result.Name)
	}

	if result.Category != CategoryRun {
		t.Errorf("Result category = %v, want %v", result.Category, CategoryRun)
	}

	expectedDesc := "Test benchmark (Docker)"
	if result.Description != expectedDesc {
		t.Errorf("Result description = %v, want %v", result.Description, expectedDesc)
	}

	if result.Iterations != 1 {
		t.Errorf("Result iterations = %v, want 1", result.Iterations)
	}

	// Verify timing is recorded
	if result.TotalTime.Duration <= 0 {
		t.Error("Result should have positive total time")
	}

	// For successful mock execution, result should succeed
	if !result.Success {
		t.Errorf("Result should be successful with mock docker, got error: %v", result.Error)
	}
}

// TestRunPodmanEquivalent tests Podman equivalent execution with mocking
func TestRunPodmanEquivalent(t *testing.T) {
	// Create a temporary directory with a mock podman command
	mockPath := createMockPodman(t)
	defer os.RemoveAll(mockPath)

	// Set PATH to use mock podman
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", mockPath+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	benchmark := Benchmark{
		Name:        "test_benchmark",
		Description: "Test benchmark",
		Category:    CategoryRun,
	}

	config := BenchmarkConfig{
		Image:       "alpine:latest",
		Command:     "echo 'test'",
		Environment: map[string]string{"TEST_VAR": "test_value"},
	}

	result := runPodmanEquivalent(benchmark, config)

	// Verify result structure
	if result.Name != "test_benchmark_podman" {
		t.Errorf("Result name = %v, want test_benchmark_podman", result.Name)
	}

	expectedDesc := "Test benchmark (Podman)"
	if result.Description != expectedDesc {
		t.Errorf("Result description = %v, want %v", result.Description, expectedDesc)
	}

	// For successful mock execution, result should succeed
	if !result.Success {
		t.Errorf("Result should be successful with mock podman, got error: %v", result.Error)
	}
}

// TestHandlingMissingTools tests behavior when Docker/Podman are not available
func TestHandlingMissingTools(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	benchmarks := []Benchmark{
		{
			Name:        "test_benchmark",
			Description: "Test benchmark",
			Category:    CategoryRun,
		},
	}

	t.Run("missing docker", func(t *testing.T) {
		// Set empty PATH to simulate docker not being available
		os.Setenv("PATH", "")

		_, err := RunDockerComparison(benchmarks)
		if err == nil {
			t.Error("Expected error when docker is not available")
		}

		if !strings.Contains(err.Error(), "docker not found") {
			t.Errorf("Error should mention docker not found, got: %v", err)
		}
	})

	t.Run("missing podman", func(t *testing.T) {
		// Set empty PATH to simulate podman not being available
		os.Setenv("PATH", "")

		_, err := RunPodmanComparison(benchmarks)
		if err == nil {
			t.Error("Expected error when podman is not available")
		}

		if !strings.Contains(err.Error(), "podman not found") {
			t.Errorf("Error should mention podman not found, got: %v", err)
		}
	})
}

// TestComparisonWithComplexConfiguration tests comparison with complex benchmark configurations
func TestComparisonWithComplexConfiguration(t *testing.T) {
	// Create a temporary directory with mock tools
	dockerMockPath := createMockDocker(t)
	defer os.RemoveAll(dockerMockPath)

	podmanMockPath := createMockPodman(t)
	defer os.RemoveAll(podmanMockPath)

	// Set PATH to use mock tools
	originalPath := os.Getenv("PATH")
	mockPath := dockerMockPath + ":" + podmanMockPath + ":" + originalPath
	os.Setenv("PATH", mockPath)
	defer os.Setenv("PATH", originalPath)

	benchmarks := []Benchmark{
		{
			Name:        "complex_benchmark",
			Description: "Complex benchmark with volumes and env vars",
			Category:    CategoryBuild,
		},
	}

	t.Run("docker with complex config", func(t *testing.T) {
		results, err := RunDockerComparison(benchmarks)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		result := results[0]
		if !strings.HasSuffix(result.Name, "_docker") {
			t.Errorf("Result name should end with _docker, got %v", result.Name)
		}

		if !result.Success {
			t.Errorf("Complex benchmark should succeed with mock docker")
		}
	})

	t.Run("podman with complex config", func(t *testing.T) {
		results, err := RunPodmanComparison(benchmarks)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		result := results[0]
		if !strings.HasSuffix(result.Name, "_podman") {
			t.Errorf("Result name should end with _podman, got %v", result.Name)
		}

		if !result.Success {
			t.Errorf("Complex benchmark should succeed with mock podman")
		}
	})
}

// TestConcurrentComparisons tests running Docker and Podman comparisons concurrently
func TestConcurrentComparisons(t *testing.T) {
	// Create mock tools
	dockerMockPath := createMockDocker(t)
	defer os.RemoveAll(dockerMockPath)

	podmanMockPath := createMockPodman(t)
	defer os.RemoveAll(podmanMockPath)

	// Set PATH to use mock tools
	originalPath := os.Getenv("PATH")
	mockPath := dockerMockPath + ":" + podmanMockPath + ":" + originalPath
	os.Setenv("PATH", mockPath)
	defer os.Setenv("PATH", originalPath)

	benchmarks := []Benchmark{
		{Name: "bench1", Description: "First benchmark", Category: CategoryRun},
		{Name: "bench2", Description: "Second benchmark", Category: CategoryBuild},
		{Name: "bench3", Description: "Third benchmark", Category: CategoryCache},
	}

	// Run comparisons concurrently
	dockerChan := make(chan []Result, 1)
	podmanChan := make(chan []Result, 1)
	dockerErrChan := make(chan error, 1)
	podmanErrChan := make(chan error, 1)

	go func() {
		results, err := RunDockerComparison(benchmarks)
		dockerChan <- results
		dockerErrChan <- err
	}()

	go func() {
		results, err := RunPodmanComparison(benchmarks)
		podmanChan <- results
		podmanErrChan <- err
	}()

	// Collect results
	dockerResults := <-dockerChan
	dockerErr := <-dockerErrChan
	podmanResults := <-podmanChan
	podmanErr := <-podmanErrChan

	// Verify both comparisons succeeded
	if dockerErr != nil {
		t.Errorf("Docker comparison failed: %v", dockerErr)
	}
	if podmanErr != nil {
		t.Errorf("Podman comparison failed: %v", podmanErr)
	}

	// Verify result counts
	if len(dockerResults) != len(benchmarks) {
		t.Errorf("Expected %d docker results, got %d", len(benchmarks), len(dockerResults))
	}
	if len(podmanResults) != len(benchmarks) {
		t.Errorf("Expected %d podman results, got %d", len(benchmarks), len(podmanResults))
	}

	// Verify naming conventions
	for i, result := range dockerResults {
		expectedName := benchmarks[i].Name + "_docker"
		if result.Name != expectedName {
			t.Errorf("Docker result[%d] name = %v, want %v", i, result.Name, expectedName)
		}
	}

	for i, result := range podmanResults {
		expectedName := benchmarks[i].Name + "_podman"
		if result.Name != expectedName {
			t.Errorf("Podman result[%d] name = %v, want %v", i, result.Name, expectedName)
		}
	}
}

// BenchmarkDockerComparison benchmarks Docker comparison performance
func BenchmarkDockerComparison(b *testing.B) {
	// Create mock docker
	mockPath := createMockDocker(b)
	defer os.RemoveAll(mockPath)

	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", mockPath+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	benchmarks := []Benchmark{
		{Name: "bench1", Description: "Benchmark 1", Category: CategoryRun},
		{Name: "bench2", Description: "Benchmark 2", Category: CategoryBuild},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := RunDockerComparison(benchmarks)
		if err != nil {
			b.Fatalf("Docker comparison failed: %v", err)
		}
	}
}

// BenchmarkPodmanComparison benchmarks Podman comparison performance
func BenchmarkPodmanComparison(b *testing.B) {
	// Create mock podman
	mockPath := createMockPodman(b)
	defer os.RemoveAll(mockPath)

	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", mockPath+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	benchmarks := []Benchmark{
		{Name: "bench1", Description: "Benchmark 1", Category: CategoryRun},
		{Name: "bench2", Description: "Benchmark 2", Category: CategoryBuild},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := RunPodmanComparison(benchmarks)
		if err != nil {
			b.Fatalf("Podman comparison failed: %v", err)
		}
	}
}

// Helper functions for creating mock tools

// createMockDocker creates a temporary directory with a mock docker executable
func createMockDocker(t testing.TB) string {
	tmpDir, err := os.MkdirTemp("", "mock-docker-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dockerScript := `#!/bin/bash
# Mock docker script
case "$1" in
    "run")
        # Simulate docker run command
        sleep 0.01  # Small delay to simulate execution time
        echo "mock docker output"
        exit 0
        ;;
    "version")
        echo "Docker version 20.10.0, build fake"
        exit 0
        ;;
    *)
        echo "mock docker: unknown command $1"
        exit 0
        ;;
esac
`

	dockerPath := tmpDir + "/docker"
	if err := os.WriteFile(dockerPath, []byte(dockerScript), 0755); err != nil {
		t.Fatalf("Failed to create mock docker: %v", err)
	}

	return tmpDir
}

// createMockPodman creates a temporary directory with a mock podman executable
func createMockPodman(t testing.TB) string {
	tmpDir, err := os.MkdirTemp("", "mock-podman-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	podmanScript := `#!/bin/bash
# Mock podman script
case "$1" in
    "run")
        # Simulate podman run command
        sleep 0.01  # Small delay to simulate execution time
        echo "mock podman output"
        exit 0
        ;;
    "version")
        echo "podman version 3.0.0"
        exit 0
        ;;
    *)
        echo "mock podman: unknown command $1"
        exit 0
        ;;
esac
`

	podmanPath := tmpDir + "/podman"
	if err := os.WriteFile(podmanPath, []byte(podmanScript), 0755); err != nil {
		t.Fatalf("Failed to create mock podman: %v", err)
	}

	return tmpDir
}

// TestMockToolsCreation tests that our mock tools are created correctly
func TestMockToolsCreation(t *testing.T) {
	t.Run("mock docker creation", func(t *testing.T) {
		mockPath := createMockDocker(t)
		defer os.RemoveAll(mockPath)

		// Test that docker executable exists and is executable
		dockerPath := mockPath + "/docker"
		info, err := os.Stat(dockerPath)
		if err != nil {
			t.Fatalf("Mock docker not created: %v", err)
		}

		if info.Mode()&0111 == 0 {
			t.Error("Mock docker is not executable")
		}

		// Test that mock docker responds to version command
		cmd := exec.Command(dockerPath, "version")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Mock docker version failed: %v", err)
		}

		if !strings.Contains(string(output), "Docker version") {
			t.Errorf("Mock docker version output unexpected: %s", output)
		}
	})

	t.Run("mock podman creation", func(t *testing.T) {
		mockPath := createMockPodman(t)
		defer os.RemoveAll(mockPath)

		// Test that podman executable exists and is executable
		podmanPath := mockPath + "/podman"
		info, err := os.Stat(podmanPath)
		if err != nil {
			t.Fatalf("Mock podman not created: %v", err)
		}

		if info.Mode()&0111 == 0 {
			t.Error("Mock podman is not executable")
		}

		// Test that mock podman responds to version command
		cmd := exec.Command(podmanPath, "version")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Mock podman version failed: %v", err)
		}

		if !strings.Contains(string(output), "podman version") {
			t.Errorf("Mock podman version output unexpected: %s", output)
		}
	})
}

// Helper function to create a valid baseline result for testing
func createValidResult() Result {
	return Result{
		Name:       "test_benchmark",
		Success:    true,
		Category:   CategoryBuild,
		Iterations: 10,
		TotalTime: Duration{
			Duration: 100 * time.Millisecond,
		},
		Min: Duration{
			Duration: 80 * time.Millisecond,
		},
		Max: Duration{
			Duration: 120 * time.Millisecond,
		},
		Mean: Duration{
			Duration: 100 * time.Millisecond,
		},
		StdDev: Duration{
			Duration: 10 * time.Millisecond,
		},
		P95: Duration{
			Duration: 115 * time.Millisecond,
		},
		P99: Duration{
			Duration: 118 * time.Millisecond,
		},
		Timestamp: time.Now().Add(-time.Minute), // 1 minute ago
	}
}
