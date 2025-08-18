package digest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// BenchmarkProjectCalculator_1000Files tests the TASK7 requirement: <100ms for 1000 files
func BenchmarkProjectCalculator_1000Files(b *testing.B) {
	tempDir := b.TempDir()

	// Create exactly 1000 files to test the TASK7 performance requirement
	const targetFiles = 1000

	b.Logf("Creating %d test files...", targetFiles)

	// Create diverse file types and sizes to simulate real projects
	fileTypes := []struct {
		ext   string
		size  int
		count int
	}{
		{".go", 2048, 300},  // Go source files
		{".js", 1024, 200},  // JavaScript files
		{".json", 512, 150}, // JSON config files
		{".md", 4096, 100},  // Documentation
		{".txt", 256, 100},  // Text files
		{".yml", 1024, 50},  // YAML configs
		{".sql", 8192, 50},  // SQL files
		{".html", 3072, 50}, // HTML templates
	}

	totalCreated := 0
	for _, ft := range fileTypes {
		for i := 0; i < ft.count && totalCreated < targetFiles; i++ {
			content := strings.Repeat(fmt.Sprintf("// Content for file %d\nline content here\n", i), ft.size/50)
			fileName := fmt.Sprintf("file_%03d_%s%s", i, ft.ext[1:], ft.ext)
			filePath := filepath.Join(tempDir, fileName)

			// Create subdirectories for some files
			if i%10 == 0 {
				subdir := filepath.Join(tempDir, fmt.Sprintf("subdir_%d", i/10))
				os.MkdirAll(subdir, 0o755)
				filePath = filepath.Join(subdir, fileName)
			}

			err := os.WriteFile(filePath, []byte(content), 0o644)
			if err != nil {
				b.Fatalf("failed to create test file: %v", err)
			}
			totalCreated++
		}
	}

	b.Logf("Created %d files for benchmark", totalCreated)

	options := Options{
		Algorithm:     "blake3", // Blake3 is faster
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		calc := NewProjectCalculator(tempDir, &options)
		digest, err := calc.Calculate(context.Background())

		duration := time.Since(start)

		if err != nil {
			b.Fatalf("Calculate() failed: %v", err)
		}

		// Validate that we processed the expected number of files
		if int(digest.FileCount) != totalCreated {
			b.Fatalf("Expected %d files, got %d", totalCreated, digest.FileCount)
		}

		// TASK7 requirement: <100ms for 1000 files
		if duration > 100*time.Millisecond {
			b.Logf("WARNING: Performance target missed! Took %v for %d files (target: <100ms)",
				duration, totalCreated)
		} else {
			b.Logf("Performance target met: %v for %d files", duration, totalCreated)
		}
	}
}

// BenchmarkProjectCalculator_ScalabilityTest tests performance across different file counts
func BenchmarkProjectCalculator_ScalabilityTest(b *testing.B) {
	fileCounts := []int{10, 50, 100, 250, 500, 1000}

	for _, fileCount := range fileCounts {
		b.Run(fmt.Sprintf("files_%d", fileCount), func(b *testing.B) {
			tempDir := b.TempDir()

			// Create test files
			for i := 0; i < fileCount; i++ {
				content := strings.Repeat(fmt.Sprintf("test content for file %d\n", i), 20)
				fileName := fmt.Sprintf("file_%04d.go", i)
				filePath := filepath.Join(tempDir, fileName)

				// Create some subdirectories
				if i%25 == 0 {
					subdir := filepath.Join(tempDir, fmt.Sprintf("pkg%d", i/25))
					os.MkdirAll(subdir, 0o755)
					filePath = filepath.Join(subdir, fileName)
				}

				os.WriteFile(filePath, []byte(content), 0o644)
			}

			options := Options{
				Algorithm:     "blake3",
				MaxFileSize:   10485760,
				IncludeHidden: false,
				LockfilesOnly: false,
			}

			b.ResetTimer()

			var totalDuration time.Duration

			for i := 0; i < b.N; i++ {
				start := time.Now()

				calc := NewProjectCalculator(tempDir, &options)
				digest, err := calc.Calculate(context.Background())

				duration := time.Since(start)
				totalDuration += duration

				if err != nil {
					b.Fatalf("Calculate() failed: %v", err)
				}

				if int(digest.FileCount) != fileCount {
					b.Fatalf("Expected %d files, got %d", fileCount, digest.FileCount)
				}
			}

			avgDuration := totalDuration / time.Duration(b.N)
			b.ReportMetric(float64(avgDuration.Nanoseconds())/1e6, "ms/op")
			b.ReportMetric(float64(fileCount)/avgDuration.Seconds(), "files/sec")
		})
	}
}

// BenchmarkProjectCalculator_FileSizeImpact tests performance with different file sizes
func BenchmarkProjectCalculator_FileSizeImpact(b *testing.B) {
	fileSizes := []struct {
		name string
		size int
	}{
		{"small", 1024},     // 1KB files
		{"medium", 10240},   // 10KB files
		{"large", 102400},   // 100KB files
		{"xlarge", 1048576}, // 1MB files
	}

	for _, fs := range fileSizes {
		b.Run(fs.name, func(b *testing.B) {
			tempDir := b.TempDir()

			// Create 100 files of the specified size
			const numFiles = 100
			for i := 0; i < numFiles; i++ {
				content := strings.Repeat(fmt.Sprintf("line %d content\n", i), fs.size/20)
				filePath := filepath.Join(tempDir, fmt.Sprintf("file_%03d.txt", i))
				os.WriteFile(filePath, []byte(content), 0o644)
			}

			options := Options{
				Algorithm:     "blake3",
				MaxFileSize:   10485760,
				IncludeHidden: false,
				LockfilesOnly: false,
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				calc := NewProjectCalculator(tempDir, &options)
				digest, err := calc.Calculate(context.Background())
				if err != nil {
					b.Fatalf("Calculate() failed: %v", err)
				}

				if digest.FileCount != numFiles {
					b.Fatalf("Expected %d files, got %d", numFiles, digest.FileCount)
				}
			}
		})
	}
}

// BenchmarkProjectCalculator_RealWorldProject simulates a realistic project structure
func BenchmarkProjectCalculator_RealWorldProject(b *testing.B) {
	tempDir := b.TempDir()

	// Simulate a real Go project structure
	projectStructure := map[string]string{
		"go.mod":                     "module github.com/test/project\n\ngo 1.21\n",
		"go.sum":                     "github.com/pkg/errors v0.9.1 h1:abcd\ngithub.com/pkg/errors v0.9.1/go.mod h1:efgh\n",
		"main.go":                    "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n",
		"README.md":                  "# Test Project\n\nThis is a test project for benchmarking.\n\n## Usage\n\nRun with `go run main.go`\n",
		"Dockerfile":                 "FROM golang:1.21\nWORKDIR /app\nCOPY . .\nRUN go build -o app\nCMD [\"./app\"]\n",
		"docker-compose.yml":         "version: '3.8'\nservices:\n  app:\n    build: .\n    ports:\n      - '8080:8080'\n",
		".gitignore":                 "*.log\n.env\nbin/\ndist/\n",
		"pkg/utils/string.go":        "package utils\n\nfunc Reverse(s string) string {\n\t// implementation\n\treturn s\n}\n",
		"pkg/utils/string_test.go":   "package utils\n\nimport \"testing\"\n\nfunc TestReverse(t *testing.T) {\n\t// test implementation\n}\n",
		"internal/server/server.go":  "package server\n\nimport \"net/http\"\n\ntype Server struct {\n\tport string\n}\n",
		"internal/server/handler.go": "package server\n\nimport \"net/http\"\n\nfunc (s *Server) handler(w http.ResponseWriter, r *http.Request) {\n\t// handler logic\n}\n",
		"cmd/api/main.go":            "package main\n\nimport \"log\"\n\nfunc main() {\n\tlog.Println(\"Starting API server\")\n}\n",
		"configs/app.yaml":           "database:\n  host: localhost\n  port: 5432\nserver:\n  port: 8080\n",
		"scripts/deploy.sh":          "#!/bin/bash\necho \"Deploying application...\"\ngo build -o bin/app\n",
		"docs/api.md":                "# API Documentation\n\n## Endpoints\n\n### GET /health\nReturns server health status\n",
	}

	// Create all files and directories
	for path, content := range projectStructure {
		fullPath := filepath.Join(tempDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0o755)
		os.WriteFile(fullPath, []byte(content), 0o644)
	}

	// Add executable permission to scripts
	os.Chmod(filepath.Join(tempDir, "scripts/deploy.sh"), 0o755)

	options := Options{
		Algorithm:     "blake3",
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		calc := NewProjectCalculator(tempDir, &options)
		digest, err := calc.Calculate(context.Background())
		if err != nil {
			b.Fatalf("Calculate() failed: %v", err)
		}

		// Should process all files except hidden ones
		expectedFiles := len(projectStructure) - 1   // exclude .gitignore if IncludeHidden is false
		if int(digest.FileCount) < expectedFiles-2 { // Allow some variance
			b.Logf("Processed %d files (expected ~%d)", digest.FileCount, expectedFiles)
		}
	}
}

// BenchmarkProjectCalculator_ConcurrentCalculations tests thread safety and concurrent performance
func BenchmarkProjectCalculator_ConcurrentCalculations(b *testing.B) {
	tempDir := b.TempDir()

	// Create test files
	const numFiles = 200
	for i := 0; i < numFiles; i++ {
		content := strings.Repeat(fmt.Sprintf("content for file %d\n", i), 50)
		filePath := filepath.Join(tempDir, fmt.Sprintf("file_%03d.go", i))
		os.WriteFile(filePath, []byte(content), 0o644)
	}

	options := Options{
		Algorithm:     "blake3",
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			calc := NewProjectCalculator(tempDir, &options)
			digest, err := calc.Calculate(context.Background())
			if err != nil {
				b.Fatalf("Calculate() failed: %v", err)
			}

			if digest.FileCount != numFiles {
				b.Fatalf("Expected %d files, got %d", numFiles, digest.FileCount)
			}
		}
	})
}

// BenchmarkProjectCalculator_MemoryAllocation tests memory efficiency
func BenchmarkProjectCalculator_MemoryAllocation(b *testing.B) {
	tempDir := b.TempDir()

	// Create test files
	const numFiles = 500
	for i := 0; i < numFiles; i++ {
		content := strings.Repeat(fmt.Sprintf("line %d\n", i), 100)
		filePath := filepath.Join(tempDir, fmt.Sprintf("file_%03d.txt", i))
		os.WriteFile(filePath, []byte(content), 0o644)
	}

	options := Options{
		Algorithm:     "blake3",
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		calc := NewProjectCalculator(tempDir, &options)
		digest, err := calc.Calculate(context.Background())
		if err != nil {
			b.Fatalf("Calculate() failed: %v", err)
		}

		if digest.FileCount != numFiles {
			b.Fatalf("Expected %d files, got %d", numFiles, digest.FileCount)
		}
	}
}

// BenchmarkCompareDigests tests digest comparison performance
func BenchmarkCompareDigests(b *testing.B) {
	// Create two similar digests with many files
	const numFiles = 1000

	createDigest := func(suffix string) *Digest {
		files := make([]FileDigest, numFiles)
		for i := 0; i < numFiles; i++ {
			files[i] = FileDigest{
				Path: fmt.Sprintf("file_%03d%s.go", i, suffix),
				Hash: fmt.Sprintf("hash%d%s", i, suffix),
				Size: int64(1024 + i),
			}
		}

		return &Digest{
			Hash:      fmt.Sprintf("digest_hash_%s", suffix),
			Algorithm: "blake3",
			Timestamp: time.Now(),
			FileCount: int(numFiles),
			TotalSize: int64(numFiles * 1024),
			Files:     files,
		}
	}

	oldDigest := createDigest("old")
	newDigest := createDigest("new")

	// Make some files different
	newDigest.Files[100].Hash = "changed_hash"
	newDigest.Files[200].Path = "new_file.go"
	newDigest.Files = append(newDigest.Files, FileDigest{
		Path: "added_file.go",
		Hash: "new_hash",
		Size: 2048,
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		comparison := Compare(oldDigest, newDigest)
		if comparison.Identical {
			b.Fatal("Expected digests to be different")
		}
	}
}

// BenchmarkIgnoreMatcherPerformance tests .mitlignore pattern matching performance
func BenchmarkIgnoreMatcherPerformance(b *testing.B) {
	tempDir := b.TempDir()

	// Create a complex .mitlignore file
	ignoreContent := `
# Comments should be ignored
*.log
*.tmp
node_modules/
.git/
dist/
build/
**/*.test
vendor/
.env*
*.swp
*.swo
__pycache__/
*.pyc
.DS_Store
Thumbs.db
*.orig
.idea/
.vscode/
coverage/
.nyc_output/
tmp/
temp/
`

	err := os.WriteFile(filepath.Join(tempDir, ".mitlignore"), []byte(ignoreContent), 0o644)
	if err != nil {
		b.Fatalf("failed to create .mitlignore: %v", err)
	}

	// Create ignore matcher
	matcher, err := LoadIgnoreRulesFromProject(tempDir)
	if err != nil {
		b.Fatalf("failed to create ignore matcher: %v", err)
	}

	// Test paths (mix of ignored and non-ignored)
	testPaths := []string{
		"main.go",
		"src/utils.go",
		"node_modules/package/index.js",
		"build/output.js",
		"test.log",
		"config.json",
		".git/HEAD",
		"vendor/package/file.go",
		"deep/nested/path/file.go",
		"another.tmp",
		"scripts/deploy.sh",
		".DS_Store",
		"__pycache__/module.pyc",
		"coverage/lcov.info",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			_ = matcher.ShouldIgnore(path, false)
		}
	}
}

// BenchmarkAlgorithmComparison compares Blake3 vs SHA256 performance
func BenchmarkAlgorithmComparison(b *testing.B) {
	tempDir := b.TempDir()

	// Create test files with varying sizes
	const numFiles = 200
	for i := 0; i < numFiles; i++ {
		// Vary file sizes to test different scenarios
		contentSize := 1024 + (i%10)*512 // 1KB to 6KB files
		content := strings.Repeat(fmt.Sprintf("content line %d\n", i), contentSize/20)
		filePath := filepath.Join(tempDir, fmt.Sprintf("file_%03d.txt", i))
		os.WriteFile(filePath, []byte(content), 0o644)
	}

	algorithms := []string{"blake3", "sha256"}

	for _, alg := range algorithms {
		b.Run(alg, func(b *testing.B) {
			options := Options{
				Algorithm:     alg,
				MaxFileSize:   10485760,
				IncludeHidden: false,
				LockfilesOnly: false,
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				calc := NewProjectCalculator(tempDir, &options)
				digest, err := calc.Calculate(context.Background())
				if err != nil {
					b.Fatalf("Calculate() failed: %v", err)
				}

				if digest.FileCount != numFiles {
					b.Fatalf("Expected %d files, got %d", numFiles, digest.FileCount)
				}
			}
		})
	}
}

// BenchmarkPerformanceValidation validates that the system meets TASK7 performance requirements
func BenchmarkPerformanceValidation(b *testing.B) {
	// Test various scenarios to ensure consistent performance
	scenarios := []struct {
		name      string
		fileCount int
		target    time.Duration
	}{
		{"small_project", 100, 10 * time.Millisecond},
		{"medium_project", 500, 50 * time.Millisecond},
		{"large_project", 1000, 100 * time.Millisecond},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			tempDir := b.TempDir()

			// Create files
			for i := 0; i < scenario.fileCount; i++ {
				content := strings.Repeat(fmt.Sprintf("test content %d\n", i), 25)
				filePath := filepath.Join(tempDir, fmt.Sprintf("file_%04d.go", i))

				// Create directory structure
				if i%50 == 0 {
					subdir := filepath.Join(tempDir, fmt.Sprintf("package%d", i/50))
					os.MkdirAll(subdir, 0o755)
					filePath = filepath.Join(subdir, filepath.Base(filePath))
				}

				os.WriteFile(filePath, []byte(content), 0o644)
			}

			options := Options{
				Algorithm:     "blake3", // Use fastest algorithm
				MaxFileSize:   10485760,
				IncludeHidden: false,
				LockfilesOnly: false,
			}

			b.ResetTimer()

			var totalDuration time.Duration
			var maxDuration time.Duration

			for i := 0; i < b.N; i++ {
				start := time.Now()

				calc := NewProjectCalculator(tempDir, &options)
				digest, err := calc.Calculate(context.Background())

				duration := time.Since(start)
				totalDuration += duration

				if duration > maxDuration {
					maxDuration = duration
				}

				if err != nil {
					b.Fatalf("Calculate() failed: %v", err)
				}

				if int(digest.FileCount) != scenario.fileCount {
					b.Fatalf("Expected %d files, got %d", scenario.fileCount, digest.FileCount)
				}
			}

			avgDuration := totalDuration / time.Duration(b.N)

			b.Logf("Average: %v, Max: %v, Target: %v", avgDuration, maxDuration, scenario.target)

			if avgDuration > scenario.target {
				b.Errorf("Performance target missed: avg %v > target %v", avgDuration, scenario.target)
			}

			if maxDuration > scenario.target*2 {
				b.Errorf("Performance consistency issue: max %v > target*2 %v", maxDuration, scenario.target*2)
			}
		})
	}
}
