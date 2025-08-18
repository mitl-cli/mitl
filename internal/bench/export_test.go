package bench

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Helper function to create test results for export testing
func createExportTestResults() []Result {
	return []Result{
		{
			Name:        "test_benchmark_1",
			Category:    CategoryBuild,
			Description: "Test benchmark 1",
			Success:     true,
			Iterations:  10,
			Mean: Duration{
				Duration: 100 * time.Millisecond,
			},
			Median: Duration{
				Duration: 95 * time.Millisecond,
			},
			Min: Duration{
				Duration: 80 * time.Millisecond,
			},
			Max: Duration{
				Duration: 120 * time.Millisecond,
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
			TotalTime: Duration{
				Duration: 1000 * time.Millisecond,
			},
			Timestamp: time.Now(),
			Memory: MemoryStats{
				AllocBytes:      1024,
				TotalAllocBytes: 2048,
			},
			Error: "",
		},
		{
			Name:        "test_benchmark_2",
			Category:    CategoryRun,
			Description: "Test benchmark 2",
			Success:     false,
			Iterations:  5,
			Mean: Duration{
				Duration: 200 * time.Millisecond,
			},
			Median: Duration{
				Duration: 190 * time.Millisecond,
			},
			Min: Duration{
				Duration: 150 * time.Millisecond,
			},
			Max: Duration{
				Duration: 250 * time.Millisecond,
			},
			StdDev: Duration{
				Duration: 25 * time.Millisecond,
			},
			P95: Duration{
				Duration: 240 * time.Millisecond,
			},
			P99: Duration{
				Duration: 245 * time.Millisecond,
			},
			TotalTime: Duration{
				Duration: 1000 * time.Millisecond,
			},
			Timestamp: time.Now(),
			Memory: MemoryStats{
				AllocBytes:      2048,
				TotalAllocBytes: 4096,
			},
			Error: "benchmark failed",
		},
	}
}

// Mock result with additional timing fields for testing
func createExtendedResult() Result {
	result := createValidResult()
	result.Median = Duration{
		Duration: 95 * time.Millisecond,
	}
	return result
}

// Helper function to create temporary test directory
func createTempTestDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "export_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})
	return tempDir
}

func TestDefaultExportOptions(t *testing.T) {
	opts := DefaultExportOptions()

	if !opts.IncludeSystemInfo {
		t.Errorf("Expected IncludeSystemInfo to be true by default")
	}
	if !opts.PrettyPrint {
		t.Errorf("Expected PrettyPrint to be true by default")
	}
	if !opts.IncludeMemory {
		t.Errorf("Expected IncludeMemory to be true by default")
	}
	if !opts.Timestamp {
		t.Errorf("Expected Timestamp to be true by default")
	}
}

func TestJSONExporter_Export(t *testing.T) {
	results := createExportTestResults()
	tempDir := createTempTestDir(t)

	tests := []struct {
		name        string
		options     ExportOptions
		filename    string
		expectError bool
	}{
		{
			name:        "export with default options",
			options:     DefaultExportOptions(),
			filename:    "test_default.json",
			expectError: false,
		},
		{
			name: "export without pretty print",
			options: ExportOptions{
				IncludeSystemInfo: true,
				PrettyPrint:       false,
				IncludeMemory:     true,
				Timestamp:         true,
			},
			filename:    "test_compact.json",
			expectError: false,
		},
		{
			name: "export minimal options",
			options: ExportOptions{
				IncludeSystemInfo: false,
				PrettyPrint:       true,
				IncludeMemory:     false,
				Timestamp:         false,
			},
			filename:    "test_minimal.json",
			expectError: false,
		},
		{
			name:        "export to nested directory",
			options:     DefaultExportOptions(),
			filename:    "nested/dir/test.json",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := NewJSONExporter()
			exporter.Options = tt.options

			path := filepath.Join(tempDir, tt.filename)
			err := exporter.Export(results, path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify file exists and contains valid JSON
			if !fileExists(path) {
				t.Errorf("Expected file %s to exist", path)
				return
			}

			// Read and parse JSON
			data, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("Failed to read file: %v", err)
				return
			}

			var doc ExportDocument
			if err := json.Unmarshal(data, &doc); err != nil {
				t.Errorf("Failed to parse JSON: %v", err)
				return
			}

			// Verify content
			if len(doc.Results) != len(results) {
				t.Errorf("Expected %d results, got %d", len(results), len(doc.Results))
			}

			if doc.Metadata.ExportFormat != "json" {
				t.Errorf("Expected export format 'json', got '%s'", doc.Metadata.ExportFormat)
			}

			if doc.Metadata.ResultCount != len(results) {
				t.Errorf("Expected result count %d, got %d", len(results), doc.Metadata.ResultCount)
			}

			// Check system info inclusion
			if tt.options.IncludeSystemInfo {
				if doc.Metadata.SystemInfo.OS == "" {
					t.Errorf("Expected system info to be included")
				}
			}
		})
	}
}

func TestJSONExporter_CreateExportDocument(t *testing.T) {
	exporter := NewJSONExporter()
	results := createExportTestResults()

	// Test with system info
	exporter.Options.IncludeSystemInfo = true
	doc := exporter.createExportDocument(results)

	if doc.Metadata.ExportFormat != "json" {
		t.Errorf("Expected export format 'json', got '%s'", doc.Metadata.ExportFormat)
	}

	if doc.Metadata.ResultCount != len(results) {
		t.Errorf("Expected result count %d, got %d", len(results), doc.Metadata.ResultCount)
	}

	if doc.Metadata.SystemInfo.OS == "" {
		t.Errorf("Expected system info to be populated")
	}

	if len(doc.Results) != len(results) {
		t.Errorf("Expected %d results, got %d", len(results), len(doc.Results))
	}

	// Test without system info
	exporter.Options.IncludeSystemInfo = false
	doc = exporter.createExportDocument(results)

	if doc.Metadata.SystemInfo.OS != "" {
		t.Errorf("Expected system info to be empty when disabled")
	}
}

func TestCSVExporter_Export(t *testing.T) {
	results := createExportTestResults()
	tempDir := createTempTestDir(t)

	tests := []struct {
		name        string
		options     ExportOptions
		filename    string
		expectError bool
	}{
		{
			name:        "export with default options",
			options:     DefaultExportOptions(),
			filename:    "test_default.csv",
			expectError: false,
		},
		{
			name: "export without memory",
			options: ExportOptions{
				IncludeSystemInfo: true,
				PrettyPrint:       true,
				IncludeMemory:     false,
				Timestamp:         true,
			},
			filename:    "test_no_memory.csv",
			expectError: false,
		},
		{
			name: "export without timestamp",
			options: ExportOptions{
				IncludeSystemInfo: false,
				PrettyPrint:       true,
				IncludeMemory:     true,
				Timestamp:         false,
			},
			filename:    "test_no_timestamp.csv",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := NewCSVExporter()
			exporter.Options = tt.options

			path := filepath.Join(tempDir, tt.filename)
			err := exporter.Export(results, path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify file exists
			if !fileExists(path) {
				t.Errorf("Expected file %s to exist", path)
				return
			}

			// Read and parse CSV
			file, err := os.Open(path)
			if err != nil {
				t.Errorf("Failed to open file: %v", err)
				return
			}
			defer file.Close()

			reader := csv.NewReader(file)
			records, err := reader.ReadAll()
			if err != nil {
				t.Errorf("Failed to read CSV: %v", err)
				return
			}

			// Verify structure
			if len(records) < 1 {
				t.Errorf("Expected at least header row")
				return
			}

			headers := records[0]
			expectedColumns := 15 // Base columns
			if tt.options.IncludeMemory {
				expectedColumns += 8 // Memory columns
			}

			if len(headers) != expectedColumns {
				t.Errorf("Expected %d columns, got %d", expectedColumns, len(headers))
			}

			// Verify data rows (skip header)
			dataRows := records[1:]
			if len(dataRows) != len(results) {
				t.Errorf("Expected %d data rows, got %d", len(results), len(dataRows))
			}

			// Verify first result data
			if len(dataRows) > 0 {
				row := dataRows[0]
				if row[0] != results[0].Name {
					t.Errorf("Expected name '%s', got '%s'", results[0].Name, row[0])
				}
			}
		})
	}
}

func TestCSVExporter_GetCSVHeaders(t *testing.T) {
	exporter := NewCSVExporter()

	// Test with memory
	exporter.Options.IncludeMemory = true
	headers := exporter.getCSVHeaders()

	expectedHeaders := []string{
		"name", "category", "description", "iterations",
		"mean_ns", "median_ns", "min_ns", "max_ns", "std_dev_ns",
		"p95_ns", "p99_ns", "total_time_ns", "success", "error", "timestamp",
		"alloc_bytes", "total_alloc_bytes", "sys_bytes", "num_gc",
		"heap_alloc_bytes", "heap_sys_bytes", "heap_idle_bytes", "heap_inuse_bytes",
	}

	if len(headers) != len(expectedHeaders) {
		t.Errorf("Expected %d headers, got %d", len(expectedHeaders), len(headers))
	}

	for i, expected := range expectedHeaders {
		if i < len(headers) && headers[i] != expected {
			t.Errorf("Expected header[%d] = '%s', got '%s'", i, expected, headers[i])
		}
	}

	// Test without memory
	exporter.Options.IncludeMemory = false
	headers = exporter.getCSVHeaders()

	if len(headers) != 15 { // Base headers only
		t.Errorf("Expected 15 base headers, got %d", len(headers))
	}
}

func TestCSVExporter_ResultToCSVRow(t *testing.T) {
	exporter := NewCSVExporter()
	result := createExportTestResults()[0]

	// Test with memory
	exporter.Options.IncludeMemory = true
    row := exporter.resultToCSVRow(&result)

	expectedLength := 23 // Base + memory columns
	if len(row) != expectedLength {
		t.Errorf("Expected row length %d, got %d", expectedLength, len(row))
	}

	// Verify specific values
	if row[0] != result.Name {
		t.Errorf("Expected name '%s', got '%s'", result.Name, row[0])
	}

	if row[1] != string(result.Category) {
		t.Errorf("Expected category '%s', got '%s'", result.Category, row[1])
	}

	if row[12] != "true" { // Success field
		t.Errorf("Expected success 'true', got '%s'", row[12])
	}

	// Test without memory
	exporter.Options.IncludeMemory = false
    row = exporter.resultToCSVRow(&result)

	if len(row) != 15 { // Base columns only
		t.Errorf("Expected row length 15, got %d", len(row))
	}
}

func TestMarkdownExporter_Export(t *testing.T) {
	results := createExportTestResults()
	tempDir := createTempTestDir(t)

	tests := []struct {
		name        string
		options     ExportOptions
		filename    string
		expectError bool
	}{
		{
			name:        "export with default options",
			options:     DefaultExportOptions(),
			filename:    "test_default.md",
			expectError: false,
		},
		{
			name: "export without system info",
			options: ExportOptions{
				IncludeSystemInfo: false,
				PrettyPrint:       true,
				IncludeMemory:     true,
				Timestamp:         true,
			},
			filename:    "test_no_sysinfo.md",
			expectError: false,
		},
		{
			name: "export without memory",
			options: ExportOptions{
				IncludeSystemInfo: true,
				PrettyPrint:       true,
				IncludeMemory:     false,
				Timestamp:         false,
			},
			filename:    "test_no_memory.md",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := NewMarkdownExporter()
			exporter.Options = tt.options

			path := filepath.Join(tempDir, tt.filename)
			err := exporter.Export(results, path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify file exists
			if !fileExists(path) {
				t.Errorf("Expected file %s to exist", path)
				return
			}

			// Read and verify content
			content, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("Failed to read file: %v", err)
				return
			}

			contentStr := string(content)

			// Verify structure
			expectedContains := []string{
				"# Benchmark Results",
				"## Results",
				"| Benchmark | Category | Mean |",
				"|-----------|----------|------|",
				results[0].Name,
				results[1].Name,
			}

			for _, expected := range expectedContains {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("Expected content to contain '%s'", expected)
				}
			}

			// Check optional sections
			if tt.options.IncludeSystemInfo {
				if !strings.Contains(contentStr, "## System Information") {
					t.Errorf("Expected system information section")
				}
			}

			if tt.options.IncludeMemory {
				if !strings.Contains(contentStr, "## Memory Usage") {
					t.Errorf("Expected memory usage section")
				}
			}

			if tt.options.Timestamp {
				if !strings.Contains(contentStr, "Generated at:") {
					t.Errorf("Expected timestamp")
				}
			}

			// Verify comparison section for multiple results
			if len(results) > 1 {
				if !strings.Contains(contentStr, "## Performance Comparison") {
					t.Errorf("Expected performance comparison section")
				}
			}
		})
	}
}

func TestHTMLExporter_Export(t *testing.T) {
	results := createExportTestResults()
	tempDir := createTempTestDir(t)

	tests := []struct {
		name        string
		options     ExportOptions
		filename    string
		expectError bool
	}{
		{
			name:        "export with default options",
			options:     DefaultExportOptions(),
			filename:    "test_default.html",
			expectError: false,
		},
		{
			name: "export minimal options",
			options: ExportOptions{
				IncludeSystemInfo: false,
				PrettyPrint:       true,
				IncludeMemory:     false,
				Timestamp:         false,
			},
			filename:    "test_minimal.html",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := NewHTMLExporter()
			exporter.Options = tt.options

			path := filepath.Join(tempDir, tt.filename)
			err := exporter.Export(results, path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify file exists
			if !fileExists(path) {
				t.Errorf("Expected file %s to exist", path)
				return
			}

			// Read and verify content
			content, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("Failed to read file: %v", err)
				return
			}

			contentStr := string(content)

			// Verify HTML structure
			expectedContains := []string{
				"<!DOCTYPE html>",
				"<html lang=\"en\">",
				"<title>Benchmark Results</title>",
				"<h1>Benchmark Results</h1>",
				"<table>",
				"<canvas id=\"benchmarkChart\">",
				results[0].Name,
				results[1].Name,
			}

			for _, expected := range expectedContains {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("Expected content to contain '%s'", expected)
				}
			}

			// Check conditional sections
			if tt.options.IncludeSystemInfo {
				if !strings.Contains(contentStr, "System Information") {
					t.Errorf("Expected system information section")
				}
			}

			if tt.options.Timestamp {
				if !strings.Contains(contentStr, "Generated:") {
					t.Errorf("Expected timestamp")
				}
			}
		})
	}
}

func TestHTMLExporter_GenerateChartData(t *testing.T) {
	exporter := NewHTMLExporter()
	results := createExportTestResults()

	chartData := exporter.generateChartData(results)

	// Verify format
	expectedContains := []string{
		"labels: [",
		"data: [",
		"'test_benchmark_1'",
		"'test_benchmark_2'",
		"100.00", // 100ms in milliseconds
		"200.00", // 200ms in milliseconds
	}

	for _, expected := range expectedContains {
		if !strings.Contains(chartData, expected) {
			t.Errorf("Expected chart data to contain '%s', got: %s", expected, chartData)
		}
	}

	// Test empty results
	emptyData := exporter.generateChartData([]Result{})
	if emptyData != "{labels: [], data: []}" {
		t.Errorf("Expected empty chart data, got: %s", emptyData)
	}
}

func TestExportToFormat(t *testing.T) {
	results := createExportTestResults()
	tempDir := createTempTestDir(t)

	tests := []struct {
		name        string
		format      string
		filename    string
		expectError bool
	}{
		{
			name:        "export to JSON",
			format:      "json",
			filename:    "test.json",
			expectError: false,
		},
		{
			name:        "export to CSV",
			format:      "csv",
			filename:    "test.csv",
			expectError: false,
		},
		{
			name:        "export to Markdown",
			format:      "markdown",
			filename:    "test.md",
			expectError: false,
		},
		{
			name:        "export to Markdown (short)",
			format:      "md",
			filename:    "test_short.md",
			expectError: false,
		},
		{
			name:        "export to HTML",
			format:      "html",
			filename:    "test.html",
			expectError: false,
		},
		{
			name:        "unsupported format",
			format:      "xml",
			filename:    "test.xml",
			expectError: true,
		},
		{
			name:        "case insensitive format",
			format:      "JSON",
			filename:    "test_upper.json",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tempDir, tt.filename)
			err := ExportToFormat(results, path, tt.format)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify file exists
			if !fileExists(path) {
				t.Errorf("Expected file %s to exist", path)
			}
		})
	}
}

func TestExportComparison(t *testing.T) {
	results1 := createExportTestResults()
	results2 := []Result{
		{
			Name:     "benchmark_3",
			Category: CategoryCache,
			Success:  true,
			Mean: Duration{
				Duration: 150 * time.Millisecond,
			},
			Timestamp: time.Now(),
		},
	}

	resultSets := map[string][]Result{
		"set1": results1,
		"set2": results2,
	}

	tempDir := createTempTestDir(t)
	path := filepath.Join(tempDir, "comparison.json")

	err := ExportComparison(resultSets, path, "json")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Verify file exists
	if !fileExists(path) {
		t.Errorf("Expected file %s to exist", path)
		return
	}

	// Read and verify content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
		return
	}

	var doc ExportDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
		return
	}

	// Verify prefixed names
	expectedNames := []string{
		"set1_test_benchmark_1",
		"set1_test_benchmark_2",
		"set2_benchmark_3",
	}

	if len(doc.Results) != len(expectedNames) {
		t.Errorf("Expected %d results, got %d", len(expectedNames), len(doc.Results))
	}

	for i, expected := range expectedNames {
		if i < len(doc.Results) && doc.Results[i].Name != expected {
			t.Errorf("Expected name '%s', got '%s'", expected, doc.Results[i].Name)
		}
	}
}

func TestEnsureDir(t *testing.T) {
	tempDir := createTempTestDir(t)

	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "create single directory",
			path:        filepath.Join(tempDir, "single", "test.txt"),
			expectError: false,
		},
		{
			name:        "create nested directories",
			path:        filepath.Join(tempDir, "nested", "deep", "path", "test.txt"),
			expectError: false,
		},
		{
			name:        "existing directory",
			path:        filepath.Join(tempDir, "test.txt"),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureDir(tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify directory exists
			dir := filepath.Dir(tt.path)
			if !dirExists(dir) {
				t.Errorf("Expected directory %s to exist", dir)
			}
		})
	}
}

func TestGetSystemInfo(t *testing.T) {
	sysInfo := getSystemInfo()

	if sysInfo.OS == "" {
		t.Errorf("Expected OS to be populated")
	}

	if sysInfo.Architecture == "" {
		t.Errorf("Expected Architecture to be populated")
	}

	if sysInfo.CPUs <= 0 {
		t.Errorf("Expected CPUs to be positive, got %d", sysInfo.CPUs)
	}

	if sysInfo.GoVersion == "" {
		t.Errorf("Expected GoVersion to be populated")
	}

	// Verify version format
	if !strings.HasPrefix(sysInfo.GoVersion, "go") {
		t.Errorf("Expected GoVersion to start with 'go', got '%s'", sysInfo.GoVersion)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			bytes:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
		{
			name:     "megabytes",
			bytes:    2 * 1024 * 1024, // 2 MB
			expected: "2.0 MB",
		},
		{
			name:     "gigabytes",
			bytes:    3 * 1024 * 1024 * 1024, // 3 GB
			expected: "3.0 GB",
		},
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "exactly 1 KB",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "large value",
			bytes:    1024 * 1024 * 1024 * 1024, // 1 TB
			expected: "1.0 TB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Test file writing permissions and error conditions
func TestExportErrorConditions(t *testing.T) {
	results := createExportTestResults()

	tests := []struct {
		name     string
		exporter Exporter
		path     string
	}{
		{
			name:     "JSON exporter invalid path",
			exporter: NewJSONExporter(),
			path:     "/invalid/path/test.json",
		},
		{
			name:     "CSV exporter invalid path",
			exporter: NewCSVExporter(),
			path:     "/invalid/path/test.csv",
		},
		{
			name:     "Markdown exporter invalid path",
			exporter: NewMarkdownExporter(),
			path:     "/invalid/path/test.md",
		},
		{
			name:     "HTML exporter invalid path",
			exporter: NewHTMLExporter(),
			path:     "/invalid/path/test.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.exporter.Export(results, tt.path)
			if err == nil {
				t.Errorf("Expected error for invalid path but got none")
			}
		})
	}
}

// Test empty results handling
func TestExportEmptyResults(t *testing.T) {
	tempDir := createTempTestDir(t)
	emptyResults := []Result{}

	exporters := map[string]Exporter{
		"json":     NewJSONExporter(),
		"csv":      NewCSVExporter(),
		"markdown": NewMarkdownExporter(),
		"html":     NewHTMLExporter(),
	}

	for format, exporter := range exporters {
		t.Run(format, func(t *testing.T) {
			path := filepath.Join(tempDir, "empty."+format)
			err := exporter.Export(emptyResults, path)
			if err != nil {
				t.Errorf("Unexpected error for empty results: %v", err)
				return
			}

			// Verify file exists
			if !fileExists(path) {
				t.Errorf("Expected file %s to exist", path)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkJSONExport(b *testing.B) {
	exporter := NewJSONExporter()
	results := createExportTestResults()
	tempDir, _ := os.MkdirTemp("", "benchmark_*")
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tempDir, "bench.json")
		_ = exporter.Export(results, path)
		os.Remove(path) // Cleanup for next iteration
	}
}

func BenchmarkCSVExport(b *testing.B) {
	exporter := NewCSVExporter()
	results := createExportTestResults()
	tempDir, _ := os.MkdirTemp("", "benchmark_*")
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tempDir, "bench.csv")
		_ = exporter.Export(results, path)
		os.Remove(path) // Cleanup for next iteration
	}
}

func BenchmarkMarkdownExport(b *testing.B) {
	exporter := NewMarkdownExporter()
	results := createExportTestResults()
	tempDir, _ := os.MkdirTemp("", "benchmark_*")
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tempDir, "bench.md")
		_ = exporter.Export(results, path)
		os.Remove(path) // Cleanup for next iteration
	}
}

func BenchmarkHTMLExport(b *testing.B) {
	exporter := NewHTMLExporter()
	results := createExportTestResults()
	tempDir, _ := os.MkdirTemp("", "benchmark_*")
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tempDir, "bench.html")
		_ = exporter.Export(results, path)
		os.Remove(path) // Cleanup for next iteration
	}
}

func BenchmarkFormatBytes(b *testing.B) {
	sizes := []uint64{
		512,
		1024,
		1024 * 1024,
		1024 * 1024 * 1024,
		1024 * 1024 * 1024 * 1024,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, size := range sizes {
			_ = formatBytes(size)
		}
	}
}

// Test concurrent exports to ensure thread safety
func TestConcurrentExports(t *testing.T) {
	results := createExportTestResults()
	tempDir := createTempTestDir(t)

	done := make(chan bool, 10)

	// Run multiple goroutines exporting simultaneously
	for i := 0; i < 10; i++ {
		go func(id int) {
			exporter := NewJSONExporter()
			path := filepath.Join(tempDir, strings.Replace("concurrent_ID.json", "ID", string(rune('0'+id)), 1))
			err := exporter.Export(results, path)
			if err != nil {
				t.Errorf("Concurrent export %d failed: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Helper functions
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}
