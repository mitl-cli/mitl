package bench

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"mitl/internal/cache"
	"mitl/internal/container"
)

// CacheBenchmark benchmarks cache scenarios including cold cache, warm cache, and cache invalidation
type CacheBenchmark struct {
	projectPath string
	clearCache  bool
	layers      int
	iterations  int
	manager     *cache.Manager
	runtime     string
	// Test fixtures
	testProjects []string
}

// NewCacheBenchmark creates a new cache benchmark with the specified configuration
func NewCacheBenchmark(projectPath string, clearCache bool, layers, iterations int) *CacheBenchmark {
    containerMgr := container.NewManager()
    rt := containerMgr.SelectOptimal()

	return &CacheBenchmark{
		projectPath: projectPath,
		clearCache:  clearCache,
		layers:      layers,
		iterations:  iterations,
        manager:     cache.NewManager(rt),
        runtime:     rt,
		testProjects: []string{
			"fixtures/node_simple",
			"fixtures/php_node",
		},
	}
}

// NewColdCacheBenchmark creates a benchmark for cold cache scenarios (no existing cache)
func NewColdCacheBenchmark(iterations int) *CacheBenchmark {
	return NewCacheBenchmark("fixtures/node_simple", true, 3, iterations)
}

// NewWarmCacheBenchmark creates a benchmark for warm cache scenarios (cache exists)
func NewWarmCacheBenchmark(iterations int) *CacheBenchmark {
	return NewCacheBenchmark("fixtures/node_simple", false, 3, iterations)
}

// NewCacheInvalidationBenchmark creates a benchmark for cache invalidation scenarios
func NewCacheInvalidationBenchmark(iterations int) *CacheBenchmark {
	return &CacheBenchmark{
		projectPath:  "fixtures/node_simple",
		clearCache:   true, // Force invalidation between runs
		layers:       5,
		iterations:   iterations,
		manager:      cache.NewManager(container.NewManager().SelectOptimal()),
		runtime:      container.NewManager().SelectOptimal(),
		testProjects: []string{"fixtures/node_simple", "fixtures/php_node"},
	}
}

// Setup prepares the benchmark for execution
func (c *CacheBenchmark) Setup() error {
	// Verify project path exists
	if _, err := os.Stat(c.projectPath); os.IsNotExist(err) {
		return fmt.Errorf("project path does not exist: %s", c.projectPath)
	}

	// Verify container runtime is available
	containerMgr := container.NewManager()
	runtimes := containerMgr.GetAvailableRuntimes()
	if len(runtimes) == 0 {
		return fmt.Errorf("no container runtimes available")
	}

	// If we're testing cold cache, clear all caches before starting
	if c.clearCache {
		if err := c.manager.ClearAll(); err != nil {
			// Don't fail if cache clear fails, just log it
			fmt.Printf("Warning: failed to clear cache: %v\n", err)
		}
	}

	return nil
}

// Run executes one iteration of the cache benchmark
func (c *CacheBenchmark) Run() (Result, error) {
	result := Result{
		Name:        c.getTestName(),
		Category:    CategoryCache,
		Description: c.getTestDescription(),
		Timestamp:   time.Now(),
		Success:     false,
	}

	// Collect memory stats before benchmark
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Start timing the cache operation
	startTime := time.Now()

	// Execute cache benchmark based on configuration
	cacheStats, err := c.executeCacheBenchmark()
	if err != nil {
		result.Error = fmt.Sprintf("cache benchmark failed: %v", err)
		result.TotalTime = Duration{time.Since(startTime)}
		return result, nil
	}

	duration := time.Since(startTime)

	// Collect memory stats after benchmark
	var memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

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

	// Store cache metrics in result metadata
	result.TotalTime = Duration{duration}
	result.Mean = Duration{duration}
	result.Success = true

	// Store cache-specific metrics (would be read by analysis tools)
	fmt.Printf("Cache benchmark completed - Hit rate: %.2f%%, Size: %d bytes, Time saved: %s\n",
		cacheStats.HitRate, cacheStats.SizeBytes, cacheStats.TimeSaved.String())

	return result, nil
}

// cacheMetrics holds cache performance metrics
type cacheMetrics struct {
	HitRate    float64
	SizeBytes  int64
	TimeSaved  time.Duration
	Operations int
}

// executeCacheBenchmark performs the actual cache benchmark operations
func (c *CacheBenchmark) executeCacheBenchmark() (cacheMetrics, error) {
	metrics := cacheMetrics{}

	// Generate test capsule tags
	baseTags := []string{
		"mitl-cache-test-node",
		"mitl-cache-test-php",
		"mitl-cache-test-multi",
	}

	hits := 0
	totalOps := 0
	var totalTimeSaved time.Duration

	for i, baseTag := range baseTags {
		if i >= c.layers {
			break
		}

		tag := fmt.Sprintf("%s-%d", baseTag, i)
		capsuleCache := c.manager.GetCapsuleCache(tag)

		// If clearing cache, invalidate this specific cache entry
		if c.clearCache {
			capsuleCache.InvalidateCache()
		}

		// First check - this will populate cache if empty
		start := time.Now()
		exists1, err := capsuleCache.Exists()
		firstCheckTime := time.Since(start)
		if err != nil {
			return metrics, fmt.Errorf("first cache check failed: %w", err)
		}

		totalOps++

		// Second check - this should hit cache
		start = time.Now()
		exists2, err := capsuleCache.Exists()
		secondCheckTime := time.Since(start)
		if err != nil {
			return metrics, fmt.Errorf("second cache check failed: %w", err)
		}

		totalOps++

		// If second check was faster, it's likely a cache hit
		if secondCheckTime < firstCheckTime && exists1 == exists2 {
			hits++
			totalTimeSaved += firstCheckTime - secondCheckTime
		}

		// Test cache with details for more comprehensive metrics
		start = time.Now()
		existsWithDetails, details, err := capsuleCache.ExistsWithDetails()
		if err != nil {
			// This might fail if image doesn't exist, which is fine for benchmarking
			fmt.Printf("Cache details check failed for %s: %v\n", tag, err)
		} else if existsWithDetails {
			metrics.SizeBytes += details.Size
		}

		totalOps++

		// Simulate cache invalidation scenario
		if c.clearCache && i%2 == 0 {
			capsuleCache.InvalidateCache()

			// Re-check after invalidation
			start = time.Now()
			_, err = capsuleCache.Exists()
			invalidationCheckTime := time.Since(start)
			if err != nil {
				return metrics, fmt.Errorf("post-invalidation cache check failed: %w", err)
			}
			totalOps++

			// Invalidation should force a fresh check (slower)
			fmt.Printf("Cache invalidation test - check time: %s\n", invalidationCheckTime.String())
		}
	}

	// Calculate hit rate
	if totalOps > 0 {
		metrics.HitRate = float64(hits) / float64(totalOps) * 100.0
	}
	metrics.Operations = totalOps
	metrics.TimeSaved = totalTimeSaved

	return metrics, nil
}

// Cleanup performs post-benchmark cleanup
func (c *CacheBenchmark) Cleanup() error {
	// Clean up any test images created during benchmarking
	// This is best effort - don't fail if cleanup fails
	if err := c.manager.ClearOld(time.Hour); err != nil {
		fmt.Printf("Warning: cache cleanup failed: %v\n", err)
	}
	return nil
}

// Iterations returns the number of iterations to run
func (c *CacheBenchmark) Iterations() int {
	if c.iterations <= 0 {
		return 5 // Default iterations for cache benchmarks
	}
	return c.iterations
}

// getTestName returns the benchmark name based on configuration
func (c *CacheBenchmark) getTestName() string {
	if c.clearCache {
		return "cache_cold_scenario"
	}
	return "cache_warm_scenario"
}

// getTestDescription returns a description of what this benchmark tests
func (c *CacheBenchmark) getTestDescription() string {
	if c.clearCache {
		return "Benchmark cache performance with cold cache (no existing cache entries)"
	}
	return "Benchmark cache performance with warm cache (existing cache entries)"
}
