package bench

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"mitl/internal/container"
)

// BuildBenchmark benchmarks container build operations with different scenarios
type BuildBenchmark struct {
	projectPath string
	dockerfile  string
	useCache    bool
	iterations  int
	manager     *container.Manager
}

// NewBuildBenchmark creates a new build benchmark with the specified configuration
func NewBuildBenchmark(projectPath, dockerfile string, useCache bool, iterations int) *BuildBenchmark {
	return &BuildBenchmark{
		projectPath: projectPath,
		dockerfile:  dockerfile,
		useCache:    useCache,
		iterations:  iterations,
		manager:     container.NewManager(),
	}
}

// NewSimpleBuildBenchmark creates a build benchmark for the node_simple fixture
func NewSimpleBuildBenchmark(iterations int) *BuildBenchmark {
	return NewBuildBenchmark("../../fixtures/node_simple", "", true, iterations)
}

// NewMultiStageBuildBenchmark creates a build benchmark for multi-stage builds
func NewMultiStageBuildBenchmark(iterations int) *BuildBenchmark {
	dockerfile := `FROM node:20-alpine AS deps
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

FROM node:20-alpine AS runner
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
EXPOSE 3000
CMD ["npm", "start"]`

	return &BuildBenchmark{
		projectPath: "../../fixtures/node_simple", // Fixed relative path from internal/bench/
		dockerfile:  dockerfile,
		useCache:    true,
		iterations:  iterations,
		manager:     container.NewManager(),
	}
}

// NewLargeDependencyBuildBenchmark creates a build benchmark for large dependency installation
func NewLargeDependencyBuildBenchmark(iterations int) *BuildBenchmark {
	dockerfile := `FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 3000
CMD ["npm", "start"]`

	return &BuildBenchmark{
		projectPath: "../../fixtures/node_simple", // Fixed relative path from internal/bench/
		dockerfile:  dockerfile,
		useCache:    false,
		iterations:  iterations,
		manager:     container.NewManager(),
	}
}

// Setup prepares the benchmark for execution
func (b *BuildBenchmark) Setup() error {
	// Verify project path exists
	if _, err := os.Stat(b.projectPath); os.IsNotExist(err) {
		return fmt.Errorf("project path does not exist: %s", b.projectPath)
	}

	// Ensure we have a container runtime
	runtimes := b.manager.GetAvailableRuntimes()
	if len(runtimes) == 0 {
		return fmt.Errorf("no container runtimes available")
	}

	return nil
}

// Run executes one iteration of the build benchmark
func (b *BuildBenchmark) Run() (Result, error) {
	result := Result{
		Name:        b.getTestName(),
		Category:    CategoryBuild,
		Description: b.getTestDescription(),
		Timestamp:   time.Now(),
		Success:     false,
	}

	// Get memory stats before build
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Generate unique tag for this iteration
	tag := fmt.Sprintf("mitl-build-bench-%d", time.Now().UnixNano())

	// Prepare build context
	buildStart := time.Now()
	err := b.executeBuild(tag)
	buildDuration := time.Since(buildStart)

	// Get memory stats after build
	var memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	// Cleanup the built image
	defer b.cleanup(tag)

	if err != nil {
		result.Error = fmt.Sprintf("build failed: %v", err)
		result.TotalTime = Duration{buildDuration}
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

	result.TotalTime = Duration{buildDuration}
	result.Mean = Duration{buildDuration}
	result.Success = true

	return result, nil
}

// executeBuild performs the actual container build operation
func (b *BuildBenchmark) executeBuild(tag string) error {
    rt := b.manager.SelectOptimal()

	var cmd *exec.Cmd

	if b.dockerfile != "" {
		// Create temporary Dockerfile
		tmpDir, err := os.MkdirTemp("", "mitl-build-bench-")
		if err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte(b.dockerfile), 0o644); err != nil {
			return fmt.Errorf("failed to write Dockerfile: %w", err)
		}

		// Copy project files to temp dir
		if err := b.copyProjectFiles(b.projectPath, tmpDir); err != nil {
			return fmt.Errorf("failed to copy project files: %w", err)
		}

		// Build with custom Dockerfile
		args := []string{"build", "-t", tag, "-f", dockerfilePath}
		if !b.useCache {
			args = append(args, "--no-cache")
		}
		args = append(args, tmpDir)

        cmd = exec.Command(rt, args...)
	} else {
		// Build with default Dockerfile in project
		args := []string{"build", "-t", tag}
		if !b.useCache {
			args = append(args, "--no-cache")
		}
		args = append(args, b.projectPath)

        cmd = exec.Command(rt, args...)
	}

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build command failed: %w, output: %s", err, string(output))
	}

	return nil
}

// copyProjectFiles copies files from source to destination directory
func (b *BuildBenchmark) copyProjectFiles(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories
		if filepath.Base(path)[0] == '.' {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}

// cleanup removes the built image
func (b *BuildBenchmark) cleanup(tag string) {
    rt := b.manager.SelectOptimal()
    cmd := exec.Command(rt, "rmi", tag)
    _ = cmd.Run() // Ignore errors during cleanup
}

// Cleanup performs post-benchmark cleanup
func (b *BuildBenchmark) Cleanup() error {
	// No persistent cleanup needed
	return nil
}

// Iterations returns the number of iterations to run
func (b *BuildBenchmark) Iterations() int {
	if b.iterations <= 0 {
		return 10 // Default iterations
	}
	return b.iterations
}

// getTestName returns the benchmark name based on configuration
func (b *BuildBenchmark) getTestName() string {
	if b.dockerfile != "" {
		if !b.useCache {
			return "build_large_dependencies"
		}
		return "build_multi_stage"
	}
	return "build_simple_node"
}

// getTestDescription returns a description of what this benchmark tests
func (b *BuildBenchmark) getTestDescription() string {
	if b.dockerfile != "" {
		if !b.useCache {
			return "Benchmark large dependency installation build without cache"
		}
		return "Benchmark multi-stage Dockerfile build with cache"
	}
	return "Benchmark simple Node.js app build with cache"
}
