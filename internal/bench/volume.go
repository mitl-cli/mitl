package bench

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"mitl/internal/container"
	"mitl/internal/volume"
)

// VolumeBenchmark benchmarks volume operations including mount performance, I/O operations, and copy operations
type VolumeBenchmark struct {
	volumePath string
	operations []string // read, write, copy
	fileSize   int64
	iterations int
	manager    *volume.Manager
	runtime    string
	// Test configuration
	fileCount   int
	tempDir     string
	testVolumes []string
}

// NewVolumeBenchmark creates a new volume benchmark with the specified configuration
func NewVolumeBenchmark(volumePath string, operations []string, fileSize int64, iterations int) *VolumeBenchmark {
	containerMgr := container.NewManager()
	runtime := containerMgr.SelectOptimal()

	return &VolumeBenchmark{
		volumePath:  volumePath,
		operations:  operations,
		fileSize:    fileSize,
		iterations:  iterations,
		manager:     volume.NewManager(runtime, ""),
		runtime:     runtime,
		fileCount:   10,
		testVolumes: []string{},
	}
}

// NewVolumeMountBenchmark creates a benchmark for volume mount/unmount performance
func NewVolumeMountBenchmark(iterations int) *VolumeBenchmark {
	return NewVolumeBenchmark("/tmp/mitl-volume-test", []string{"mount"}, 1024*1024, iterations) // 1MB files
}

// NewVolumeIOBenchmark creates a benchmark for volume I/O operations
func NewVolumeIOBenchmark(iterations int) *VolumeBenchmark {
	return NewVolumeBenchmark("/tmp/mitl-volume-test", []string{"read", "write"}, 10*1024*1024, iterations) // 10MB files
}

// NewVolumeCopyBenchmark creates a benchmark for volume copy operations
func NewVolumeCopyBenchmark(iterations int) *VolumeBenchmark {
	return NewVolumeBenchmark("/tmp/mitl-volume-test", []string{"copy"}, 5*1024*1024, iterations) // 5MB files
}

// NewLargeFileVolumeBenchmark creates a benchmark for large file operations
func NewLargeFileVolumeBenchmark(iterations int) *VolumeBenchmark {
	return &VolumeBenchmark{
		volumePath:  "/tmp/mitl-volume-large-test",
		operations:  []string{"read", "write", "copy"},
		fileSize:    100 * 1024 * 1024, // 100MB files
		iterations:  iterations,
		manager:     volume.NewManager(container.NewManager().SelectOptimal(), ""),
		runtime:     container.NewManager().SelectOptimal(),
		fileCount:   5, // Fewer large files
		testVolumes: []string{},
	}
}

// Setup prepares the benchmark for execution
func (v *VolumeBenchmark) Setup() error {
	// Verify container runtime is available
	containerMgr := container.NewManager()
	runtimes := containerMgr.GetAvailableRuntimes()
	if len(runtimes) == 0 {
		return fmt.Errorf("no container runtimes available")
	}

	// Create temporary directory for test files
	var err error
	v.tempDir, err = os.MkdirTemp("", "mitl-volume-bench-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Create test volumes for different volume types
	volumeTypes := []volume.VolumeType{
		volume.VolumeTypePnpmModules,
		volume.VolumeTypeVendor,
		volume.VolumeTypePythonVenv,
	}

	for _, vt := range volumeTypes {
		volumeName, _, err := v.manager.GetOrCreateVolume(vt, "test-hash-12345678")
		if err != nil {
			// Volume creation might fail on some systems, continue with warning
			fmt.Printf("Warning: failed to create test volume for %s: %v\n", vt, err)
			continue
		}
		v.testVolumes = append(v.testVolumes, volumeName)
	}

	// Create test files if we're testing file operations
	if v.containsOperation("read") || v.containsOperation("write") || v.containsOperation("copy") {
		if err := v.createTestFiles(); err != nil {
			return fmt.Errorf("failed to create test files: %w", err)
		}
	}

	return nil
}

// Run executes one iteration of the volume benchmark
func (v *VolumeBenchmark) Run() (Result, error) {
	result := Result{
		Name:        v.getTestName(),
		Category:    CategoryVolume,
		Description: v.getTestDescription(),
		Timestamp:   time.Now(),
		Success:     false,
	}

	// Collect memory stats before benchmark
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Start timing the volume operations
	startTime := time.Now()

	// Execute volume benchmark based on operations
	volumeStats, err := v.executeVolumeBenchmark()
	if err != nil {
		result.Error = fmt.Sprintf("volume benchmark failed: %v", err)
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

	// Store volume metrics in result
	result.TotalTime = Duration{duration}
	result.Mean = Duration{duration}
	result.Success = true

	// Store volume-specific metrics (would be read by analysis tools)
	fmt.Printf("Volume benchmark completed - Throughput: %.2f MB/s, IOPS: %.0f, Latency: %s\n",
		volumeStats.ThroughputMBs, volumeStats.IOPS, volumeStats.AvgLatency.String())

	return result, nil
}

// volumeMetrics holds volume performance metrics
type volumeMetrics struct {
	ThroughputMBs    float64
	IOPS             float64
	AvgLatency       time.Duration
	MountTime        time.Duration
	Operations       int
	BytesTransferred int64
}

// executeVolumeBenchmark performs the actual volume benchmark operations
func (v *VolumeBenchmark) executeVolumeBenchmark() (volumeMetrics, error) {
	metrics := volumeMetrics{}
	var totalLatency time.Duration
	operations := 0
	var totalBytes int64

	// Test volume mount performance
	if v.containsOperation("mount") {
		mountTime, err := v.benchmarkVolumeMount()
		if err != nil {
			return metrics, fmt.Errorf("mount benchmark failed: %w", err)
		}
		metrics.MountTime = mountTime
		fmt.Printf("Volume mount time: %s\n", mountTime.String())
	}

	// Test file read operations
	if v.containsOperation("read") {
		readLatency, readBytes, readOps, err := v.benchmarkVolumeRead()
		if err != nil {
			return metrics, fmt.Errorf("read benchmark failed: %w", err)
		}
		totalLatency += readLatency
		totalBytes += readBytes
		operations += readOps
	}

	// Test file write operations
	if v.containsOperation("write") {
		writeLatency, writeBytes, writeOps, err := v.benchmarkVolumeWrite()
		if err != nil {
			return metrics, fmt.Errorf("write benchmark failed: %w", err)
		}
		totalLatency += writeLatency
		totalBytes += writeBytes
		operations += writeOps
	}

	// Test file copy operations
	if v.containsOperation("copy") {
		copyLatency, copyBytes, copyOps, err := v.benchmarkVolumeCopy()
		if err != nil {
			return metrics, fmt.Errorf("copy benchmark failed: %w", err)
		}
		totalLatency += copyLatency
		totalBytes += copyBytes
		operations += copyOps
	}

	// Calculate metrics
	if operations > 0 {
		metrics.AvgLatency = totalLatency / time.Duration(operations)
		metrics.IOPS = float64(operations) / totalLatency.Seconds()
	}
	if totalBytes > 0 && totalLatency > 0 {
		metrics.ThroughputMBs = float64(totalBytes) / (1024 * 1024) / totalLatency.Seconds()
	}
	metrics.Operations = operations
	metrics.BytesTransferred = totalBytes

	return metrics, nil
}

// benchmarkVolumeMount measures volume mount/unmount performance
func (v *VolumeBenchmark) benchmarkVolumeMount() (time.Duration, error) {
	if len(v.testVolumes) == 0 {
		// Create a test volume for mounting
		testVolume := fmt.Sprintf("mitl-mount-test-%d", time.Now().UnixNano())
		cmd := exec.Command(v.runtime, "volume", "create", testVolume)
		if err := cmd.Run(); err != nil {
			return 0, fmt.Errorf("failed to create test volume: %w", err)
		}
		defer func() {
			_ = exec.Command(v.runtime, "volume", "rm", testVolume).Run()
		}()
		v.testVolumes = append(v.testVolumes, testVolume)
	}

	// Measure time to mount volume in a container and perform basic operation
	start := time.Now()

	containerName := fmt.Sprintf("mitl-mount-test-%d", time.Now().UnixNano())
	cmd := exec.Command(v.runtime, "run", "--rm", "--name", containerName,
		"-v", fmt.Sprintf("%s:/test-mount", v.testVolumes[0]),
		"alpine:latest", "sh", "-c", "ls /test-mount && echo 'mount-test' > /test-mount/test.txt")

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("mount test failed: %w", err)
	}

	mountTime := time.Since(start)
	return mountTime, nil
}

// benchmarkVolumeRead measures volume read performance
func (v *VolumeBenchmark) benchmarkVolumeRead() (time.Duration, int64, int, error) {
	var totalLatency time.Duration
	var totalBytes int64
	operations := 0

	// Read test files from volume
	for i := 0; i < v.fileCount; i++ {
		testFile := filepath.Join(v.tempDir, fmt.Sprintf("test-read-%d.dat", i))

		start := time.Now()
		data, err := os.ReadFile(testFile)
		latency := time.Since(start)

		if err != nil {
			return totalLatency, totalBytes, operations, fmt.Errorf("failed to read test file %s: %w", testFile, err)
		}

		totalLatency += latency
		totalBytes += int64(len(data))
		operations++
	}

	return totalLatency, totalBytes, operations, nil
}

// benchmarkVolumeWrite measures volume write performance
func (v *VolumeBenchmark) benchmarkVolumeWrite() (time.Duration, int64, int, error) {
	var totalLatency time.Duration
	var totalBytes int64
	operations := 0

	// Write test files to volume
	for i := 0; i < v.fileCount; i++ {
		testFile := filepath.Join(v.tempDir, fmt.Sprintf("test-write-%d.dat", i))
		data := make([]byte, v.fileSize)
		_, _ = rand.Read(data) // Fill with random data

		start := time.Now()
		err := os.WriteFile(testFile, data, 0644)
		latency := time.Since(start)

		if err != nil {
			return totalLatency, totalBytes, operations, fmt.Errorf("failed to write test file %s: %w", testFile, err)
		}

		totalLatency += latency
		totalBytes += int64(len(data))
		operations++
	}

	return totalLatency, totalBytes, operations, nil
}

// benchmarkVolumeCopy measures volume copy performance using container operations
func (v *VolumeBenchmark) benchmarkVolumeCopy() (time.Duration, int64, int, error) {
	var totalLatency time.Duration
	var totalBytes int64
	operations := 0

	if len(v.testVolumes) == 0 {
		return 0, 0, 0, fmt.Errorf("no test volumes available for copy benchmark")
	}

	// Copy files between volumes using container
	for i := 0; i < v.fileCount && i < len(v.testVolumes); i++ {
		containerName := fmt.Sprintf("mitl-copy-test-%d", time.Now().UnixNano())
		sourceFile := filepath.Join(v.tempDir, fmt.Sprintf("test-copy-source-%d.dat", i))

		// Create source file
		data := make([]byte, v.fileSize)
		_, _ = rand.Read(data)
		if err := os.WriteFile(sourceFile, data, 0644); err != nil {
			return totalLatency, totalBytes, operations, fmt.Errorf("failed to create source file: %w", err)
		}

		start := time.Now()

		// Copy file to volume using container
		cmd := exec.Command(v.runtime, "run", "--rm", "--name", containerName,
			"-v", fmt.Sprintf("%s:/source", v.tempDir),
			"-v", fmt.Sprintf("%s:/dest", v.testVolumes[i%len(v.testVolumes)]),
			"alpine:latest", "cp", fmt.Sprintf("/source/test-copy-source-%d.dat", i), "/dest/")

		err := cmd.Run()
		latency := time.Since(start)

		if err != nil {
			return totalLatency, totalBytes, operations, fmt.Errorf("copy operation failed: %w", err)
		}

		totalLatency += latency
		totalBytes += int64(len(data))
		operations++

		// Cleanup source file
		_ = os.Remove(sourceFile)
	}

	return totalLatency, totalBytes, operations, nil
}

// createTestFiles creates test files for benchmarking
func (v *VolumeBenchmark) createTestFiles() error {
	for i := 0; i < v.fileCount; i++ {
		testFile := filepath.Join(v.tempDir, fmt.Sprintf("test-read-%d.dat", i))
		data := make([]byte, v.fileSize)
		_, _ = rand.Read(data) // Fill with random data

		if err := os.WriteFile(testFile, data, 0644); err != nil {
			return fmt.Errorf("failed to create test file %s: %w", testFile, err)
		}
	}
	return nil
}

// containsOperation checks if the benchmark includes a specific operation
func (v *VolumeBenchmark) containsOperation(op string) bool {
	for _, operation := range v.operations {
		if operation == op {
			return true
		}
	}
	return false
}

// Cleanup performs post-benchmark cleanup
func (v *VolumeBenchmark) Cleanup() error {
	// Clean up temporary directory
	if v.tempDir != "" {
		if err := os.RemoveAll(v.tempDir); err != nil {
			fmt.Printf("Warning: failed to cleanup temp dir %s: %v\n", v.tempDir, err)
		}
	}

	// Clean up test volumes (best effort)
	for _, volumeName := range v.testVolumes {
		cmd := exec.Command(v.runtime, "volume", "rm", volumeName)
		_ = cmd.Run() // Ignore errors during cleanup
	}

	// Clean up old volumes
	if err := v.manager.CleanOldVolumes(time.Hour); err != nil {
		fmt.Printf("Warning: volume cleanup failed: %v\n", err)
	}

	return nil
}

// Iterations returns the number of iterations to run
func (v *VolumeBenchmark) Iterations() int {
	if v.iterations <= 0 {
		return 3 // Default iterations for volume benchmarks (fewer due to I/O cost)
	}
	return v.iterations
}

// getTestName returns the benchmark name based on operations
func (v *VolumeBenchmark) getTestName() string {
	if len(v.operations) == 1 {
		return fmt.Sprintf("volume_%s", v.operations[0])
	}
	return fmt.Sprintf("volume_%s", strings.Join(v.operations, "_"))
}

// getTestDescription returns a description of what this benchmark tests
func (v *VolumeBenchmark) getTestDescription() string {
	if len(v.operations) == 1 {
		switch v.operations[0] {
		case "mount":
			return "Benchmark volume mount and unmount performance"
		case "read":
			return "Benchmark volume read I/O performance and throughput"
		case "write":
			return "Benchmark volume write I/O performance and throughput"
		case "copy":
			return "Benchmark volume copy operations between containers"
		}
	}
	return fmt.Sprintf("Benchmark volume operations: %s", strings.Join(v.operations, ", "))
}
