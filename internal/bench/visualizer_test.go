package bench

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Helper function to create test results for visualization testing
func createTestResults() []Result {
	return []Result{
		{
			Name:    "mitl_build",
			Success: true,
			Mean:    Duration{Duration: 100 * time.Millisecond},
			Median:  Duration{Duration: 95 * time.Millisecond},
			Min:     Duration{Duration: 80 * time.Millisecond},
			Max:     Duration{Duration: 120 * time.Millisecond},
			StdDev:  Duration{Duration: 10 * time.Millisecond},
		},
		{
			Name:    "docker_build",
			Success: true,
			Mean:    Duration{Duration: 150 * time.Millisecond},
			Median:  Duration{Duration: 145 * time.Millisecond},
			Min:     Duration{Duration: 130 * time.Millisecond},
			Max:     Duration{Duration: 180 * time.Millisecond},
			StdDev:  Duration{Duration: 15 * time.Millisecond},
		},
		{
			Name:    "podman_build",
			Success: true,
			Mean:    Duration{Duration: 200 * time.Millisecond},
			Median:  Duration{Duration: 190 * time.Millisecond},
			Min:     Duration{Duration: 170 * time.Millisecond},
			Max:     Duration{Duration: 250 * time.Millisecond},
			StdDev:  Duration{Duration: 25 * time.Millisecond},
		},
	}
}

func TestDefaultVisualizerOptions(t *testing.T) {
	// Test without COLUMNS env var
	os.Unsetenv("COLUMNS")
	opts := DefaultVisualizerOptions()

	if opts.Width != 80 {
		t.Errorf("Expected default width 80, got %d", opts.Width)
	}
	if !opts.UseColors {
		t.Errorf("Expected UseColors to be true by default")
	}
	if !opts.ShowValues {
		t.Errorf("Expected ShowValues to be true by default")
	}

	// Test with COLUMNS env var
	os.Setenv("COLUMNS", "120")
	defer os.Unsetenv("COLUMNS")

	opts = DefaultVisualizerOptions()
	if opts.Width != 120 {
		t.Errorf("Expected width from COLUMNS env var (120), got %d", opts.Width)
	}

	// Test with invalid COLUMNS env var
	os.Setenv("COLUMNS", "invalid")
	opts = DefaultVisualizerOptions()
	if opts.Width != 80 {
		t.Errorf("Expected fallback to default width (80) with invalid COLUMNS, got %d", opts.Width)
	}
}

func TestBarChart_Render(t *testing.T) {
	tests := []struct {
		name           string
		results        []Result
		title          string
		options        VisualizerOptions
		expectContains []string
		expectNotEmpty bool
	}{
		{
			name:           "empty results",
			results:        []Result{},
			title:          "Test Chart",
			expectContains: []string{"No results to display"},
			expectNotEmpty: true,
		},
		{
			name:    "single result",
			results: createTestResults()[:1],
			title:   "Single Benchmark",
			options: VisualizerOptions{Width: 80, UseColors: false, ShowValues: true},
			expectContains: []string{
				"Single Benchmark",
				"mitl_build",
				"100.0ms",
				"█",
			},
			expectNotEmpty: true,
		},
		{
			name:    "multiple results with comparison",
			results: createTestResults(),
			title:   "Benchmark Comparison",
			options: VisualizerOptions{Width: 100, UseColors: false, ShowValues: true},
			expectContains: []string{
				"Benchmark Comparison",
				"mitl_build",
				"docker_build",
				"podman_build",
				"100.0ms",
				"150.0ms",
				"200.0ms",
				"Speed improvement",
			},
			expectNotEmpty: true,
		},
		{
			name:    "with colors enabled",
			results: createTestResults(),
			title:   "Colored Chart",
			options: VisualizerOptions{Width: 80, UseColors: true, ShowValues: true},
			expectContains: []string{
				"mitl_build",
				"docker_build",
				"podman_build",
			},
			expectNotEmpty: true,
		},
		{
			name: "zero duration results",
			results: []Result{
				{
					Name:   "zero_benchmark",
					Mean:   Duration{Duration: 0},
					Min:    Duration{Duration: 0},
					Max:    Duration{Duration: 0},
					StdDev: Duration{Duration: 0},
				},
			},
			title:          "Zero Duration",
			expectContains: []string{"No valid durations to display"},
			expectNotEmpty: true,
		},
		{
			name:    "narrow terminal width",
			results: createTestResults()[:1],
			title:   "Narrow",
			options: VisualizerOptions{Width: 20, UseColors: false, ShowValues: true},
			expectContains: []string{
				"mitl_build",
				"█",
			},
			expectNotEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chart := NewBarChart(tt.title)
			if tt.options.Width > 0 {
				chart.Options = tt.options
			}

			output := chart.Render(tt.results)

			if tt.expectNotEmpty && output == "" {
				t.Errorf("Expected non-empty output")
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestBarChart_ColorizeBar(t *testing.T) {
	chart := NewBarChart("Test")
	chart.Options.UseColors = true

	tests := []struct {
		name     string
		bar      string
		expected string
	}{
		{
			name:     "mitl_benchmark",
			bar:      "████",
			expected: "\033[32m████\033[0m", // Green
		},
		{
			name:     "docker_benchmark",
			bar:      "████",
			expected: "\033[34m████\033[0m", // Blue
		},
		{
			name:     "podman_benchmark",
			bar:      "████",
			expected: "\033[33m████\033[0m", // Yellow
		},
		{
			name:     "other_benchmark",
			bar:      "████",
			expected: "████", // No color
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chart.colorizeBar(tt.name, tt.bar)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}

	// Test with colors disabled
	chart.Options.UseColors = false
	result := chart.colorizeBar("mitl_test", "████")
	if result != "████" {
		t.Errorf("Expected no color when UseColors=false, got '%s'", result)
	}
}

func TestBarChart_CalculateLabelWidth(t *testing.T) {
	chart := NewBarChart("Test")

	tests := []struct {
		name     string
		results  []Result
		expected int
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: 2, // Just padding
		},
		{
			name: "varying name lengths",
			results: []Result{
				{Name: "short"},
				{Name: "very_long_benchmark_name"},
				{Name: "mid"},
			},
			expected: 26, // longest name (24) + padding (2)
		},
		{
			name: "single result",
			results: []Result{
				{Name: "test_benchmark"},
			},
			expected: 16, // name length (14) + padding (2)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width := chart.calculateLabelWidth(tt.results)
			if width != tt.expected {
				t.Errorf("Expected width %d, got %d", tt.expected, width)
			}
		})
	}
}

func TestSparklineChart_Render(t *testing.T) {
	tests := []struct {
		name           string
		results        []Result
		title          string
		options        VisualizerOptions
		expectContains []string
		expectNotEmpty bool
	}{
		{
			name:           "empty results",
			results:        []Result{},
			title:          "Trend",
			expectContains: []string{"No results to display"},
			expectNotEmpty: true,
		},
		{
			name:    "single result",
			results: createTestResults()[:1],
			title:   "Single Trend",
			options: VisualizerOptions{ShowValues: true},
			expectContains: []string{
				"Single Trend:",
				"▄", // Middle sparkline character for single value
				"min:",
				"max:",
			},
			expectNotEmpty: true,
		},
		{
			name:    "multiple results ascending",
			results: createTestResults(),
			title:   "Performance Trend",
			options: VisualizerOptions{ShowValues: true},
			expectContains: []string{
				"Performance Trend:",
				"min:",
				"max:",
			},
			expectNotEmpty: true,
		},
		{
			name: "identical durations",
			results: []Result{
				{Name: "test1", Mean: Duration{Duration: 100 * time.Millisecond}},
				{Name: "test2", Mean: Duration{Duration: 100 * time.Millisecond}},
				{Name: "test3", Mean: Duration{Duration: 100 * time.Millisecond}},
			},
			title:   "Stable Performance",
			options: VisualizerOptions{ShowValues: true},
			expectContains: []string{
				"Stable Performance:",
				"▄", // Middle character for equal values
			},
			expectNotEmpty: true,
		},
		{
			name:    "without values",
			results: createTestResults()[:1],
			title:   "No Values",
			options: VisualizerOptions{ShowValues: false},
			expectContains: []string{
				"No Values:",
			},
			expectNotEmpty: true,
		},
		{
			name:    "no title",
			results: createTestResults()[:1],
			title:   "",
			options: VisualizerOptions{ShowValues: false},
			expectContains: []string{
				"▄",
			},
			expectNotEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chart := NewSparklineChart(tt.title)
			if tt.options.ShowValues || !tt.options.ShowValues {
				chart.Options = tt.options
			}

			output := chart.Render(tt.results)

			if tt.expectNotEmpty && output == "" {
				t.Errorf("Expected non-empty output")
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestSparklineChart_CreateSparkline(t *testing.T) {
	chart := NewSparklineChart("Test")

	tests := []struct {
		name     string
		results  []Result
		expected string
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: "",
		},
		{
			name: "ascending values",
			results: []Result{
				{Mean: Duration{Duration: 10 * time.Millisecond}},
				{Mean: Duration{Duration: 50 * time.Millisecond}},
				{Mean: Duration{Duration: 100 * time.Millisecond}},
			},
			expected: "▁▄█", // Should map to low, mid, high
		},
		{
			name: "identical values",
			results: []Result{
				{Mean: Duration{Duration: 50 * time.Millisecond}},
				{Mean: Duration{Duration: 50 * time.Millisecond}},
			},
			expected: "▄▄", // Middle character for equal values
		},
		{
			name: "single value",
			results: []Result{
				{Mean: Duration{Duration: 50 * time.Millisecond}},
			},
			expected: "▄", // Middle character for single value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chart.createSparkline(tt.results)
			if result != tt.expected {
				t.Errorf("Expected sparkline '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSparklineChart_GetMinMaxDuration(t *testing.T) {
	chart := NewSparklineChart("Test")
	results := createTestResults()

	min := chart.getMinDuration(results)
	max := chart.getMaxDuration(results)

	expectedMin := 100 * time.Millisecond // mitl_build
	expectedMax := 200 * time.Millisecond // podman_build

	if min != expectedMin {
		t.Errorf("Expected min duration %v, got %v", expectedMin, min)
	}
	if max != expectedMax {
		t.Errorf("Expected max duration %v, got %v", expectedMax, max)
	}

	// Test empty results
	emptyMin := chart.getMinDuration([]Result{})
	emptyMax := chart.getMaxDuration([]Result{})
	if emptyMin != 0 || emptyMax != 0 {
		t.Errorf("Expected 0 duration for empty results, got min=%v, max=%v", emptyMin, emptyMax)
	}
}

func TestComparisonTable_Render(t *testing.T) {
	tests := []struct {
		name           string
		results        []Result
		title          string
		expectContains []string
		expectNotEmpty bool
	}{
		{
			name:           "empty results",
			results:        []Result{},
			title:          "Comparison",
			expectContains: []string{"No results to display"},
			expectNotEmpty: true,
		},
		{
			name:    "single result",
			results: createTestResults()[:1],
			title:   "Single Result Table",
			expectContains: []string{
				"Single Result Table",
				"Benchmark",
				"Mean",
				"Median",
				"Min",
				"Max",
				"StdDev",
				"Speedup",
				"mitl_build",
				"100.0ms",
				"95.0ms",
				"80.0ms",
				"120.0ms",
				"10.0ms",
				"1.00x",
				"|",
				"-",
			},
			expectNotEmpty: true,
		},
		{
			name:    "multiple results with speedup calculation",
			results: createTestResults(),
			title:   "Performance Comparison",
			expectContains: []string{
				"Performance Comparison",
				"mitl_build",
				"docker_build",
				"podman_build",
				"1.00x", // Fastest baseline
				"1.50x", // Docker vs mitl
				"2.00x", // Podman vs mitl
			},
			expectNotEmpty: true,
		},
		{
			name:    "no title",
			results: createTestResults()[:1],
			title:   "",
			expectContains: []string{
				"Benchmark",
				"mitl_build",
			},
			expectNotEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := NewComparisonTable(tt.title)
			output := table.Render(tt.results)

			if tt.expectNotEmpty && output == "" {
				t.Errorf("Expected non-empty output")
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestComparisonTable_CalculateColumnWidths(t *testing.T) {
	table := NewComparisonTable("Test")
	results := createTestResults()
	headers := []string{"Benchmark", "Mean", "Median", "Min", "Max", "StdDev", "Speedup"}

	widths := table.calculateColumnWidths(results, headers)

	// Check that widths are reasonable
	if len(widths) != len(headers) {
		t.Errorf("Expected %d column widths, got %d", len(headers), len(widths))
	}

	// Each width should be at least the header length + padding
	for i, header := range headers {
		if widths[i] < len(header)+2 {
			t.Errorf("Width for column '%s' (%d) is less than header length + padding (%d)",
				header, widths[i], len(header)+2)
		}
	}

	// Test with empty results
	emptyWidths := table.calculateColumnWidths([]Result{}, headers)
	for i, header := range headers {
		if emptyWidths[i] != len(header)+2 {
			t.Errorf("Expected width %d for empty results, got %d", len(header)+2, emptyWidths[i])
		}
	}
}

func TestComparisonTable_FindBaseline(t *testing.T) {
	table := NewComparisonTable("Test")
	results := createTestResults()

	baseline := table.findBaseline(results)

	// Should be mitl_build (100ms) as it's the fastest
	if baseline.Name != "mitl_build" {
		t.Errorf("Expected baseline to be 'mitl_build', got '%s'", baseline.Name)
	}
	if baseline.Mean.Duration != 100*time.Millisecond {
		t.Errorf("Expected baseline duration 100ms, got %v", baseline.Mean.Duration)
	}

	// Test with empty results
	emptyBaseline := table.findBaseline([]Result{})
	if emptyBaseline.Name != "" {
		t.Errorf("Expected empty baseline for empty results")
	}

	// Test with single result
	single := []Result{results[0]}
	singleBaseline := table.findBaseline(single)
	if singleBaseline.Name != results[0].Name {
		t.Errorf("Expected single result as baseline")
	}
}

func TestComparisonTable_CalculateSpeedup(t *testing.T) {
	table := NewComparisonTable("Test")
	results := createTestResults()
	baseline := results[0] // mitl_build, 100ms

	tests := []struct {
		name     string
		baseline Result
		result   Result
		expected string
	}{
		{
			name:     "same as baseline",
			baseline: baseline,
			result:   baseline,
			expected: "1.00x",
		},
		{
			name:     "2x slower",
			baseline: baseline,
			result:   results[2], // podman_build, 200ms
			expected: "2.00x",
		},
		{
			name:     "1.5x slower",
			baseline: baseline,
			result:   results[1], // docker_build, 150ms
			expected: "1.50x",
		},
		{
			name:     "zero baseline",
			baseline: Result{Mean: Duration{Duration: 0}},
			result:   baseline,
			expected: "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			speedup := table.calculateSpeedup(&tt.baseline, &tt.result)
			if speedup != tt.expected {
				t.Errorf("Expected speedup '%s', got '%s'", tt.expected, speedup)
			}
		})
	}
}

func TestComparisonTable_RenderTableRow(t *testing.T) {
	table := NewComparisonTable("Test")

	columns := []string{"Col1", "Col2", "Col3"}
	widths := []int{6, 8, 10}

	// Test regular row
	row := table.renderTableRow(columns, widths, false)
	expected := "| Col1 | Col2   | Col3     |\n"
	if row != expected {
		t.Errorf("Expected row '%s', got '%s'", expected, row)
	}

	// Test header row (same formatting for now)
	headerRow := table.renderTableRow(columns, widths, true)
	if headerRow != expected {
		t.Errorf("Expected header row '%s', got '%s'", expected, headerRow)
	}

	// Test with mismatched columns and widths
	shortWidths := []int{6}
	shortRow := table.renderTableRow(columns, shortWidths, false)
	expectedShort := "| Col1 |\n"
	if shortRow != expectedShort {
		t.Errorf("Expected short row '%s', got '%s'", expectedShort, shortRow)
	}
}

func TestComparisonTable_RenderSeparator(t *testing.T) {
	table := NewComparisonTable("Test")
	widths := []int{6, 8, 10}

	separator := table.renderSeparator(widths)
	expected := "|------|--------|----------|\n"
	if separator != expected {
		t.Errorf("Expected separator '%s', got '%s'", expected, separator)
	}

	// Test with empty widths
	emptySeparator := table.renderSeparator([]int{})
	if emptySeparator != "|\n" {
		t.Errorf("Expected empty separator '|\\n', got '%s'", emptySeparator)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "nanoseconds",
			duration: 500 * time.Nanosecond,
			expected: "500ns",
		},
		{
			name:     "microseconds",
			duration: 500 * time.Microsecond,
			expected: "500.0µs",
		},
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500.0ms",
		},
		{
			name:     "seconds",
			duration: 2500 * time.Millisecond,
			expected: "2.50s",
		},
		{
			name:     "zero duration",
			duration: 0,
			expected: "0ns",
		},
		{
			name:     "edge case - exactly 1µs",
			duration: time.Microsecond,
			expected: "1.0µs",
		},
		{
			name:     "edge case - exactly 1ms",
			duration: time.Millisecond,
			expected: "1.0ms",
		},
		{
			name:     "edge case - exactly 1s",
			duration: time.Second,
			expected: "1.00s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetTerminalWidth(t *testing.T) {
	// Save original env
	originalColumns := os.Getenv("COLUMNS")
	defer os.Setenv("COLUMNS", originalColumns)

	// Test default fallback
	os.Unsetenv("COLUMNS")
	width := getTerminalWidth()
	if width != 80 {
		t.Errorf("Expected default width 80, got %d", width)
	}

	// Test valid COLUMNS env var
	os.Setenv("COLUMNS", "120")
	width = getTerminalWidth()
	if width != 120 {
		t.Errorf("Expected width 120 from COLUMNS env var, got %d", width)
	}

	// Test invalid COLUMNS env var
	os.Setenv("COLUMNS", "invalid")
	width = getTerminalWidth()
	if width != 80 {
		t.Errorf("Expected default width 80 with invalid COLUMNS, got %d", width)
	}

	// Test negative COLUMNS env var
	os.Setenv("COLUMNS", "-10")
	width = getTerminalWidth()
	if width != 80 {
		t.Errorf("Expected default width 80 with negative COLUMNS, got %d", width)
	}

	// Test zero COLUMNS env var
	os.Setenv("COLUMNS", "0")
	width = getTerminalWidth()
	if width != 80 {
		t.Errorf("Expected default width 80 with zero COLUMNS, got %d", width)
	}
}

func TestIsErrorResult(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		expected bool
	}{
		{
			name: "successful result",
			result: Result{
				Success: true,
				Error:   "",
			},
			expected: false,
		},
		{
			name: "failed result",
			result: Result{
				Success: false,
				Error:   "",
			},
			expected: true,
		},
		{
			name: "result with error",
			result: Result{
				Success: true,
				Error:   "some error",
			},
			expected: true,
		},
		{
			name: "failed result with error",
			result: Result{
				Success: false,
				Error:   "some error",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isErrorResult(&tt.result)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFormatResults(t *testing.T) {
	results := createTestResults()
	title := "Quick Format Test"

	output := FormatResults(results, title)

	// Should contain title and result names
	expectedContains := []string{
		title,
		"mitl_build",
		"docker_build",
		"podman_build",
		"█", // Bar character
	}

	for _, expected := range expectedContains {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', got:\n%s", expected, output)
		}
	}
}

func TestFormatTrend(t *testing.T) {
	results := createTestResults()
	title := "Trend Test"

	output := FormatTrend(results, title)

	// Should contain title and sparkline characters
	expectedContains := []string{
		title,
		"min:",
		"max:",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', got:\n%s", expected, output)
		}
	}

	// Should contain sparkline characters
	sparklineChars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	hasSparkline := false
	for _, char := range sparklineChars {
		if strings.Contains(output, char) {
			hasSparkline = true
			break
		}
	}
	if !hasSparkline {
		t.Errorf("Expected output to contain sparkline characters, got:\n%s", output)
	}
}

func TestFormatComparison(t *testing.T) {
	results := createTestResults()
	title := "Comparison Test"

	output := FormatComparison(results, title)

	// Should contain title, headers, and table formatting
	expectedContains := []string{
		title,
		"Benchmark",
		"Mean",
		"Speedup",
		"mitl_build",
		"docker_build",
		"podman_build",
		"|", // Table formatting
		"-", // Table separator
	}

	for _, expected := range expectedContains {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', got:\n%s", expected, output)
		}
	}
}

// Benchmark tests for performance validation
func BenchmarkBarChart_Render(b *testing.B) {
	chart := NewBarChart("Performance Test")
	results := createTestResults()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chart.Render(results)
	}
}

func BenchmarkSparklineChart_Render(b *testing.B) {
	chart := NewSparklineChart("Performance Test")
	results := createTestResults()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chart.Render(results)
	}
}

func BenchmarkComparisonTable_Render(b *testing.B) {
	table := NewComparisonTable("Performance Test")
	results := createTestResults()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.Render(results)
	}
}

func BenchmarkFormatDuration(b *testing.B) {
	durations := []time.Duration{
		500 * time.Nanosecond,
		500 * time.Microsecond,
		500 * time.Millisecond,
		2500 * time.Millisecond,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, d := range durations {
			_ = formatDuration(d)
		}
	}
}

// Test edge cases and boundary conditions
func TestVisualizerEdgeCases(t *testing.T) {
	t.Run("bar chart with very long names", func(t *testing.T) {
		results := []Result{
			{
				Name: strings.Repeat("very_long_benchmark_name", 10), // 250+ chars
				Mean: Duration{Duration: 100 * time.Millisecond},
			},
		}

		chart := NewBarChart("Long Names Test")
		chart.Options.Width = 50 // Narrow width

		output := chart.Render(results)
		if output == "" {
			t.Errorf("Expected non-empty output for long names")
		}
	})

	t.Run("sparkline with extreme value differences", func(t *testing.T) {
		results := []Result{
			{Mean: Duration{Duration: 1 * time.Nanosecond}},
			{Mean: Duration{Duration: 1 * time.Hour}},
		}

		chart := NewSparklineChart("Extreme Values")
		output := chart.Render(results)

		if !strings.Contains(output, "▁") || !strings.Contains(output, "█") {
			t.Errorf("Expected extreme sparkline characters for extreme values")
		}
	})

	t.Run("comparison table with zero durations mixed", func(t *testing.T) {
		results := []Result{
			{Name: "zero", Mean: Duration{Duration: 0}},
			{Name: "normal", Mean: Duration{Duration: 100 * time.Millisecond}},
		}

		table := NewComparisonTable("Mixed Zero Values")
		output := table.Render(results)

		if !strings.Contains(output, "N/A") {
			t.Errorf("Expected N/A speedup for zero baseline")
		}
	})
}

// Test concurrent access to ensure thread safety
func TestVisualizerConcurrency(t *testing.T) {
	results := createTestResults()
	chart := NewBarChart("Concurrency Test")

	done := make(chan bool, 10)

	// Run multiple goroutines rendering simultaneously
	for i := 0; i < 10; i++ {
		go func() {
			output := chart.Render(results)
			if output == "" {
				t.Errorf("Unexpected empty output in concurrent test")
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
