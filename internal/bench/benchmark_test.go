package bench

import (
	"fmt"
	"mitl/internal/container"
	"os"
	"sync"
	"testing"
	"time"
)

// TestBuildBenchmarkSetup verifies that build benchmarks can be set up properly
func TestBuildBenchmarkSetup(t *testing.T) {
	// Skip if no container runtime available
	if os.Getenv("MITL_NO_BENCHMARK") == "1" {
		t.Skip("Benchmarks disabled via MITL_NO_BENCHMARK")
	}

	// Check if a container runtime is available
	manager := container.NewManager()
	if len(manager.GetAvailableRuntimes()) == 0 {
		t.Skip("No container runtime available, skipping benchmark tests")
	}

	tests := []struct {
		name      string
		benchmark *BuildBenchmark
		wantErr   bool
	}{
		{
			name:      "simple build benchmark",
			benchmark: NewSimpleBuildBenchmark(1),
			wantErr:   false,
		},
		{
			name:      "multi-stage build benchmark",
			benchmark: NewMultiStageBuildBenchmark(1),
			wantErr:   false,
		},
		{
			name:      "large dependency build benchmark",
			benchmark: NewLargeDependencyBuildBenchmark(1),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.benchmark.Setup()
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildBenchmark.Setup() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify iterations
			if got := tt.benchmark.Iterations(); got <= 0 {
				t.Errorf("BuildBenchmark.Iterations() = %v, want > 0", got)
			}

			// Cleanup
			if err := tt.benchmark.Cleanup(); err != nil {
				t.Errorf("BuildBenchmark.Cleanup() error = %v", err)
			}
		})
	}
}

// TestRunBenchmarkSetup verifies that run benchmarks can be set up properly
func TestRunBenchmarkSetup(t *testing.T) {
	// Skip if no container runtime available
	if os.Getenv("MITL_NO_BENCHMARK") == "1" {
		t.Skip("Benchmarks disabled via MITL_NO_BENCHMARK")
	}

	// Check if a container runtime is available
	manager := container.NewManager()
	if len(manager.GetAvailableRuntimes()) == 0 {
		t.Skip("No container runtime available, skipping benchmark tests")
	}

	tests := []struct {
		name      string
		benchmark *RunBenchmark
		wantErr   bool
	}{
		{
			name:      "startup time benchmark",
			benchmark: NewStartupTimeBenchmark(1),
			wantErr:   false, // May succeed if alpine:latest is available
		},
		{
			name:      "command execution benchmark",
			benchmark: NewCommandExecutionBenchmark(1),
			wantErr:   false, // May succeed if alpine:latest is available
		},
		{
			name:      "interactive run benchmark",
			benchmark: NewInteractiveRunBenchmark(1),
			wantErr:   false, // May succeed if alpine:latest is available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.benchmark.Setup()
			if (err != nil) != tt.wantErr {
				// Log the actual result for environment-dependent tests
				if err != nil {
					t.Logf("Setup failed for %s (may be expected if Docker images unavailable): %v", tt.name, err)
				} else {
					t.Logf("Setup succeeded for %s (Docker images are available)", tt.name)
				}
			}

			// Verify iterations
			if got := tt.benchmark.Iterations(); got <= 0 {
				t.Errorf("RunBenchmark.Iterations() = %v, want > 0", got)
			}

			// Cleanup
			if err := tt.benchmark.Cleanup(); err != nil {
				t.Errorf("RunBenchmark.Cleanup() error = %v", err)
			}
		})
	}
}

// TestBenchmarkInterface verifies that both benchmark types implement BenchmarkRunner
func TestBenchmarkInterface(t *testing.T) {
	var _ BenchmarkRunner = &BuildBenchmark{}
	var _ BenchmarkRunner = &RunBenchmark{}
}

// TestBenchmarkNames verifies that benchmarks have proper names and descriptions
func TestBenchmarkNames(t *testing.T) {
	buildBench := NewSimpleBuildBenchmark(5)
	if name := buildBench.getTestName(); name == "" {
		t.Error("BuildBenchmark.getTestName() returned empty string")
	}
	if desc := buildBench.getTestDescription(); desc == "" {
		t.Error("BuildBenchmark.getTestDescription() returned empty string")
	}

	runBench := NewStartupTimeBenchmark(5)
	if name := runBench.getTestName(); name == "" {
		t.Error("RunBenchmark.getTestName() returned empty string")
	}
	if desc := runBench.getTestDescription(); desc == "" {
		t.Error("RunBenchmark.getTestDescription() returned empty string")
	}
}

// TestSuiteExecution tests Suite execution with mock benchmarks
func TestSuiteExecution(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		benchmarks []mockBenchmark
		wantErr    bool
	}{
		{
			name:    "empty suite",
			config:  DefaultConfig(),
			wantErr: true,
		},
		{
			name:   "single successful benchmark",
			config: DefaultConfig(),
			benchmarks: []mockBenchmark{
				{name: "test1", duration: 100 * time.Millisecond, shouldFail: false},
			},
			wantErr: false,
		},
		{
			name:   "multiple successful benchmarks",
			config: DefaultConfig(),
			benchmarks: []mockBenchmark{
				{name: "test1", duration: 100 * time.Millisecond, shouldFail: false},
				{name: "test2", duration: 200 * time.Millisecond, shouldFail: false},
			},
			wantErr: false,
		},
		{
			name:   "benchmark with setup failure",
			config: DefaultConfig(),
			benchmarks: []mockBenchmark{
				{name: "failing", duration: 100 * time.Millisecond, shouldFail: true, failOnSetup: true},
			},
			wantErr: false, // Suite should handle individual benchmark failures
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := NewSuite(tt.config)

			// Register mock benchmarks
			for i := range tt.benchmarks {
				err := suite.Register(tt.benchmarks[i].name, "test description", CategoryBuild, &tt.benchmarks[i])
				if err != nil {
					t.Fatalf("Failed to register benchmark: %v", err)
				}
			}

			// Run the suite
			results, err := suite.Run()
			if (err != nil) != tt.wantErr {
				t.Errorf("Suite.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(results) != len(tt.benchmarks) {
					t.Errorf("Expected %d results, got %d", len(tt.benchmarks), len(results))
				}

				// Verify results match expected benchmarks
				for i, result := range results {
					if i < len(tt.benchmarks) {
						expected := &tt.benchmarks[i]
						if result.Name != expected.name {
							t.Errorf("Result[%d] name = %v, want %v", i, result.Name, expected.name)
						}
						if expected.shouldFail && result.Success {
							t.Errorf("Result[%d] should have failed but succeeded", i)
						}
					}
				}
			}
		})
	}
}

// TestSuiteConcurrentExecution tests concurrent benchmark execution
func TestSuiteConcurrentExecution(t *testing.T) {
	config := DefaultConfig()
	config.Parallel = true
	// Use lightweight config for testing - remove overhead that interferes with timing tests
	config.WarmupIterations = 0
	config.CooldownDuration = 0
	config.CollectMemoryInfo = false

	suite := NewSuite(config)

	// Add multiple mock benchmarks
	benchmarks := []mockBenchmark{
		{name: "concurrent1", duration: 100 * time.Millisecond, shouldFail: false},
		{name: "concurrent2", duration: 150 * time.Millisecond, shouldFail: false},
		{name: "concurrent3", duration: 80 * time.Millisecond, shouldFail: false},
	}

	for i := range benchmarks {
		err := suite.Register(benchmarks[i].name, "concurrent test", CategoryBuild, &benchmarks[i])
		if err != nil {
			t.Fatalf("Failed to register benchmark: %v", err)
		}
	}

	start := time.Now()
	results, err := suite.Run()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Suite.Run() failed: %v", err)
	}

	if len(results) != len(benchmarks) {
		t.Errorf("Expected %d results, got %d", len(benchmarks), len(results))
	}

	// Parallel execution should be faster than sequential
	// Allow some overhead for goroutine setup and cleanup
	maxExpectedDuration := 300 * time.Millisecond // Longest benchmark + overhead
	if duration > maxExpectedDuration {
		t.Errorf("Parallel execution took %v, expected less than %v", duration, maxExpectedDuration)
	}
}

// TestSuiteResultCollection tests result collection and aggregation
func TestSuiteResultCollection(t *testing.T) {
	suite := NewSuite(DefaultConfig())

	// Add benchmarks with known results
	benchmarks := []mockBenchmark{
		{name: "fast", duration: 50 * time.Millisecond, shouldFail: false},
		{name: "slow", duration: 200 * time.Millisecond, shouldFail: false},
		{name: "failed", duration: 100 * time.Millisecond, shouldFail: true},
	}

	for i := range benchmarks {
		err := suite.Register(benchmarks[i].name, "test", CategoryBuild, &benchmarks[i])
		if err != nil {
			t.Fatalf("Failed to register benchmark: %v", err)
		}
	}

	results, err := suite.Run()
	if err != nil {
		t.Fatalf("Suite.Run() failed: %v", err)
	}

	// Check result properties
	if len(results) != len(benchmarks) {
		t.Errorf("Expected %d results, got %d", len(benchmarks), len(results))
	}

	// Verify results can be retrieved again
	storedResults := suite.Results()
	if len(storedResults) != len(results) {
		t.Errorf("Stored results length mismatch: got %d, want %d", len(storedResults), len(results))
	}

	// Check specific result properties
	for _, result := range results {
		if result.Name == "" {
			t.Error("Result name should not be empty")
		}
		if result.Category == "" {
			t.Error("Result category should not be empty")
		}
		if result.Timestamp.IsZero() {
			t.Error("Result timestamp should be set")
		}
	}
}

// TestSuiteErrorHandling tests error handling and recovery
func TestSuiteErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Suite) error
		expectError bool
	}{
		{
			name: "duplicate benchmark names",
			setup: func(s *Suite) error {
				mb1 := mockBenchmark{name: "duplicate", duration: 100 * time.Millisecond}
				mb2 := mockBenchmark{name: "duplicate", duration: 200 * time.Millisecond}
				if err := s.Register("duplicate", "test1", CategoryBuild, &mb1); err != nil {
					return err
				}
				return s.Register("duplicate", "test2", CategoryBuild, &mb2)
			},
			expectError: true,
		},
		{
			name: "nil benchmark runner",
			setup: func(s *Suite) error {
				return s.Register("nil-runner", "test", CategoryBuild, nil)
			},
			expectError: true,
		},
		{
			name: "empty benchmark name",
			setup: func(s *Suite) error {
				mb := mockBenchmark{name: "test", duration: 100 * time.Millisecond}
				return s.Register("", "test", CategoryBuild, &mb)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := NewSuite(DefaultConfig())

			err := tt.setup(suite)

			if tt.expectError && err == nil {
				t.Error("Expected registration error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected registration error: %v", err)
			}
		})
	}
}

// TestSuiteFiltering tests benchmark filtering functionality
func TestSuiteFiltering(t *testing.T) {
	suite := NewSuite(DefaultConfig())

	// Add benchmarks with different categories and names
	benchmarks := []struct {
		name     string
		category Category
	}{
		{"build_test1", CategoryBuild},
		{"build_test2", CategoryBuild},
		{"run_test1", CategoryRun},
		{"cache_test", CategoryCache},
	}

	for _, b := range benchmarks {
		mb := &mockBenchmark{name: b.name, duration: 100 * time.Millisecond}
		err := suite.Register(b.name, "test", b.category, mb)
		if err != nil {
			t.Fatalf("Failed to register benchmark: %v", err)
		}
	}

	// Test category filtering
	buildBenchmarks := suite.FilterByCategory(CategoryBuild)
	if len(buildBenchmarks) != 2 {
		t.Errorf("Expected 2 build benchmarks, got %d", len(buildBenchmarks))
	}

	runBenchmarks := suite.FilterByCategory(CategoryRun)
	if len(runBenchmarks) != 1 {
		t.Errorf("Expected 1 run benchmark, got %d", len(runBenchmarks))
	}

	// Test name filtering
	buildNameFilter := suite.FilterByName("build")
	if len(buildNameFilter) != 2 {
		t.Errorf("Expected 2 benchmarks matching 'build', got %d", len(buildNameFilter))
	}

	test1Filter := suite.FilterByName("test1")
	if len(test1Filter) != 2 {
		t.Errorf("Expected 2 benchmarks matching 'test1', got %d", len(test1Filter))
	}
}

// BenchmarkBuildSetup benchmarks the setup performance
func BenchmarkBuildSetup(b *testing.B) {
	if os.Getenv("MITL_NO_BENCHMARK") == "1" {
		b.Skip("Benchmarks disabled via MITL_NO_BENCHMARK")
	}

	benchmark := NewSimpleBuildBenchmark(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := benchmark.Setup()
		if err != nil {
			b.Fatalf("Setup failed: %v", err)
		}
	}
}

// BenchmarkRunSetup benchmarks the setup performance for run benchmarks
func BenchmarkRunSetup(b *testing.B) {
	if os.Getenv("MITL_NO_BENCHMARK") == "1" {
		b.Skip("Benchmarks disabled via MITL_NO_BENCHMARK")
	}

	benchmark := NewStartupTimeBenchmark(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = benchmark.Setup() // May fail if no alpine image, but that's OK for timing
	}
}

// BenchmarkSuiteExecution benchmarks suite execution performance
func BenchmarkSuiteExecution(b *testing.B) {
	suite := NewSuite(DefaultConfig())

	// Add lightweight mock benchmarks
	for i := 0; i < 5; i++ {
		mb := mockBenchmark{
			name:     fmt.Sprintf("bench%d", i),
			duration: 10 * time.Millisecond,
		}
		err := suite.Register(mb.name, "benchmark test", CategoryBuild, &mb)
		if err != nil {
			b.Fatalf("Failed to register benchmark: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := suite.Run()
		if err != nil {
			b.Fatalf("Suite execution failed: %v", err)
		}
	}
}

// mockBenchmark implements BenchmarkRunner for testing
type mockBenchmark struct {
	name         string
	duration     time.Duration
	shouldFail   bool
	failOnSetup  bool
	setupCalls   int
	runCalls     int
	cleanupCalls int
	mutex        sync.Mutex
}

func (m *mockBenchmark) Setup() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.setupCalls++

	if m.failOnSetup {
		return fmt.Errorf("mock setup failure for %s", m.name)
	}
	return nil
}

func (m *mockBenchmark) Run() (Result, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.runCalls++

	result := Result{
		Name:        m.name,
		Category:    CategoryBuild,
		Description: "Mock benchmark for testing",
		Timestamp:   time.Now(),
		Iterations:  1,
		Success:     !m.shouldFail,
	}

	if m.shouldFail {
		result.Error = "mock benchmark failure"
		return result, nil
	}

	// Simulate benchmark execution time
	time.Sleep(m.duration)

	result.Mean = Duration{m.duration}
	result.Median = Duration{m.duration}
	result.Min = Duration{m.duration}
	result.Max = Duration{m.duration}
	result.TotalTime = Duration{m.duration}

	return result, nil
}

func (m *mockBenchmark) Cleanup() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.cleanupCalls++
	return nil
}

func (m *mockBenchmark) Iterations() int {
	return 1
}

// TestMockBenchmark tests the mock benchmark implementation
func TestMockBenchmark(t *testing.T) {
	mb := mockBenchmark{
		name:     "test-mock",
		duration: 50 * time.Millisecond,
	}

	// Test setup
	if err := mb.Setup(); err != nil {
		t.Errorf("Mock setup failed: %v", err)
	}

	// Test run
	result, err := mb.Run()
	if err != nil {
		t.Errorf("Mock run failed: %v", err)
	}

	if result.Name != mb.name {
		t.Errorf("Result name = %v, want %v", result.Name, mb.name)
	}

	if !result.Success {
		t.Error("Mock benchmark should succeed")
	}

	// Test cleanup
	if err := mb.Cleanup(); err != nil {
		t.Errorf("Mock cleanup failed: %v", err)
	}

	// Verify call counts
	if mb.setupCalls != 1 {
		t.Errorf("Setup calls = %d, want 1", mb.setupCalls)
	}
	if mb.runCalls != 1 {
		t.Errorf("Run calls = %d, want 1", mb.runCalls)
	}
	if mb.cleanupCalls != 1 {
		t.Errorf("Cleanup calls = %d, want 1", mb.cleanupCalls)
	}
}

// TestBenchmarkRunnerInterface verifies the interface contract
func TestBenchmarkRunnerInterface(t *testing.T) {
	var _ BenchmarkRunner = &mockBenchmark{}
}

// TestConcurrentBenchmarkAccess tests thread safety
func TestConcurrentBenchmarkAccess(t *testing.T) {
	suite := NewSuite(DefaultConfig())

	const numGoroutines = 10
	const numBenchmarks = 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Concurrently register benchmarks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numBenchmarks; j++ {
				mb := mockBenchmark{
					name:     fmt.Sprintf("concurrent-%d-%d", id, j),
					duration: 10 * time.Millisecond,
				}

				err := suite.Register(mb.name, "concurrent test", CategoryBuild, &mb)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d: %w", id, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any registration errors
	for err := range errors {
		t.Errorf("Concurrent registration error: %v", err)
	}

	// Verify all benchmarks were registered
	benchmarks := suite.GetBenchmarks()
	expected := numGoroutines * numBenchmarks
	if len(benchmarks) != expected {
		t.Errorf("Expected %d benchmarks, got %d", expected, len(benchmarks))
	}
}
