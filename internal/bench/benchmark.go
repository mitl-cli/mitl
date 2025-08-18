package bench

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// BenchmarkRunner defines the interface for benchmark implementations
type BenchmarkRunner interface {
	// Setup prepares the benchmark for execution
	Setup() error
	// Run executes one iteration of the benchmark and returns the result
	Run() (Result, error)
	// Cleanup performs post-benchmark cleanup
	Cleanup() error
	// Iterations returns the number of iterations to run
	Iterations() int
}

// Benchmark represents a single benchmark
type Benchmark struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    Category        `json:"category"`
	Runner      BenchmarkRunner `json:"-"`
}

// Suite manages a collection of benchmarks
type Suite struct {
	benchmarks []Benchmark
	config     Config
	results    []Result
	mutex      sync.RWMutex
}

// NewSuite creates a new benchmark suite with the given configuration
func NewSuite(config Config) *Suite {
	return &Suite{
		benchmarks: make([]Benchmark, 0),
		config:     config,
		results:    make([]Result, 0),
	}
}

// Register adds a benchmark to the suite
func (s *Suite) Register(name, description string, category Category, runner BenchmarkRunner) error {
	if name == "" {
		return fmt.Errorf("benchmark name cannot be empty")
	}
	if runner == nil {
		return fmt.Errorf("benchmark runner cannot be nil")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check for duplicate names
	for _, b := range s.benchmarks {
		if b.Name == name {
			return fmt.Errorf("benchmark with name %q already registered", name)
		}
	}

	benchmark := Benchmark{
		Name:        name,
		Description: description,
		Category:    category,
		Runner:      runner,
	}

	s.benchmarks = append(s.benchmarks, benchmark)
	return nil
}

// Run executes all registered benchmarks
func (s *Suite) Run() ([]Result, error) {
	s.mutex.Lock()
	benchmarks := make([]Benchmark, len(s.benchmarks))
	copy(benchmarks, s.benchmarks)
	s.mutex.Unlock()

	if len(benchmarks) == 0 {
		return nil, fmt.Errorf("no benchmarks registered")
	}

	results := make([]Result, 0, len(benchmarks))

	if s.config.Parallel {
		// Run benchmarks in parallel
		resultsChan := make(chan Result, len(benchmarks))
		errChan := make(chan error, len(benchmarks))
		var wg sync.WaitGroup

		for _, benchmark := range benchmarks {
			wg.Add(1)
			go func(b Benchmark) {
				defer wg.Done()
				result, err := s.runBenchmark(b)
				if err != nil {
					errChan <- fmt.Errorf("benchmark %q failed: %w", b.Name, err)
					return
				}
				resultsChan <- result
			}(benchmark)
		}

		// Wait for all goroutines to complete and close channels
		go func() {
			wg.Wait()
			close(resultsChan)
			close(errChan)
		}()

		// Collect all results first, then check for errors
		resultsCollected := 0
		expectedResults := len(benchmarks)

		for resultsCollected < expectedResults {
			select {
			case result, ok := <-resultsChan:
				if ok {
					results = append(results, result)
					resultsCollected++
				}
			case err := <-errChan:
				if err != nil {
					return nil, err
				}
			}
		}

		// Check if there are any remaining errors
		select {
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
		default:
			// No more errors
		}

		// Sort results by name for consistent output
		sort.Slice(results, func(i, j int) bool {
			return results[i].Name < results[j].Name
		})
	} else {
		// Run benchmarks sequentially
		for _, benchmark := range benchmarks {
			result, err := s.runBenchmark(benchmark)
			if err != nil {
				return nil, fmt.Errorf("benchmark %q failed: %w", benchmark.Name, err)
			}
			results = append(results, result)
		}
	}

	s.mutex.Lock()
	s.results = results
	s.mutex.Unlock()

	return results, nil
}

// runBenchmark executes a single benchmark and returns its result
func (s *Suite) runBenchmark(benchmark Benchmark) (Result, error) {
	result := Result{
		Name:        benchmark.Name,
		Category:    benchmark.Category,
		Description: benchmark.Description,
		Timestamp:   time.Now(),
		Success:     false,
	}

	// Setup
	if err := benchmark.Runner.Setup(); err != nil {
		result.Error = fmt.Sprintf("setup failed: %v", err)
		return result, nil // Return result with error, don't fail the entire suite
	}

	defer func() {
		if err := benchmark.Runner.Cleanup(); err != nil && s.config.Verbose {
			// Log cleanup errors but don't fail the benchmark
			fmt.Printf("Warning: cleanup failed for benchmark %q: %v\n", benchmark.Name, err)
		}
	}()

	// Determine number of iterations
	iterations := benchmark.Runner.Iterations()
	if iterations <= 0 {
		iterations = s.config.MinIterations
	}
	if iterations > s.config.MaxIterations {
		iterations = s.config.MaxIterations
	}

	// Warmup iterations
	for i := 0; i < s.config.WarmupIterations; i++ {
		if _, err := benchmark.Runner.Run(); err != nil {
			result.Error = fmt.Sprintf("warmup iteration %d failed: %v", i+1, err)
			return result, nil
		}
	}

	// Cooldown after warmup
	if s.config.CooldownDuration > 0 {
		time.Sleep(s.config.CooldownDuration)
	}

	// Collect memory stats before benchmark if requested
	var memBefore runtime.MemStats
	if s.config.CollectMemoryInfo {
		runtime.GC()
		runtime.ReadMemStats(&memBefore)
	}

	// Run benchmark iterations
	durations := make([]time.Duration, 0, iterations)
	startTime := time.Now()

	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		benchResult, err := benchmark.Runner.Run()
		iterDuration := time.Since(iterStart)

		if err != nil {
			result.Error = fmt.Sprintf("iteration %d failed: %v", i+1, err)
			return result, nil
		}

		// Use the duration from the benchmark result if it has one, otherwise use measured time
		if benchResult.TotalTime.Duration > 0 {
			durations = append(durations, benchResult.TotalTime.Duration)
		} else {
			durations = append(durations, iterDuration)
		}

		// Check if we've exceeded the maximum duration
		if s.config.Duration > 0 && time.Since(startTime) > s.config.Duration {
			break
		}
	}

	totalTime := time.Since(startTime)

	// Collect memory stats after benchmark if requested
	var memAfter runtime.MemStats
	if s.config.CollectMemoryInfo {
		runtime.GC()
		runtime.ReadMemStats(&memAfter)

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
	}

    // Calculate statistics
    stats := calculateStats(durations)

    result.Iterations = len(durations)
    result.Mean = Duration{stats.Mean}
    result.Median = Duration{stats.Median}
    result.Min = Duration{stats.Min}
    result.Max = Duration{stats.Max}
    result.StdDev = Duration{stats.StdDev}
    result.P95 = Duration{stats.P95}
    result.P99 = Duration{stats.P99}
	result.TotalTime = Duration{totalTime}
	result.Success = true

	return result, nil
}

// Results returns the results of the last benchmark run
func (s *Suite) Results() []Result {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	results := make([]Result, len(s.results))
	copy(results, s.results)
	return results
}

// GetBenchmarks returns a copy of all registered benchmarks
func (s *Suite) GetBenchmarks() []Benchmark {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	benchmarks := make([]Benchmark, len(s.benchmarks))
	copy(benchmarks, s.benchmarks)
	return benchmarks
}

// Clear removes all registered benchmarks and results
func (s *Suite) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.benchmarks = s.benchmarks[:0]
	s.results = s.results[:0]
}

// FilterByCategory returns benchmarks matching the given category
func (s *Suite) FilterByCategory(category Category) []Benchmark {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var filtered []Benchmark
	for _, b := range s.benchmarks {
		if b.Category == category {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// FilterByName returns benchmarks with names containing the given substring
func (s *Suite) FilterByName(nameFilter string) []Benchmark {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var filtered []Benchmark
	for _, b := range s.benchmarks {
		if strings.Contains(strings.ToLower(b.Name), strings.ToLower(nameFilter)) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}
