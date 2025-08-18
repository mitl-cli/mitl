package bench

import (
    "fmt"
    "os"
    "sort"
    "strconv"
    "strings"
    "time"
)

const (
    msgNoResults          = "No results to display"
    msgNoValidDurations   = "No valid durations to display"
)

// Visualizer provides an interface for rendering benchmark results
type Visualizer interface {
	// Render converts benchmark results into a visual string representation
	Render(results []Result) string
}

// VisualizerOptions configure the visualization output
type VisualizerOptions struct {
	Width      int  // Terminal width (default: 80)
	UseColors  bool // Enable ANSI color output
	ShowValues bool // Show numeric values alongside visualizations
}

// DefaultVisualizerOptions returns sensible defaults for visualization
func DefaultVisualizerOptions() VisualizerOptions {
	width := 80
	if w := getTerminalWidth(); w > 0 {
		width = w
	}

	return VisualizerOptions{
		Width:      width,
		UseColors:  true,
		ShowValues: true,
	}
}

// BarChart creates ASCII bar charts for benchmark comparisons
type BarChart struct {
	Options VisualizerOptions
	Title   string
}

// NewBarChart creates a new bar chart visualizer
func NewBarChart(title string) *BarChart {
	return &BarChart{
		Options: DefaultVisualizerOptions(),
		Title:   title,
	}
}

// Render creates an ASCII bar chart from benchmark results
func (bc *BarChart) Render(results []Result) string {
    if len(results) == 0 {
        return msgNoResults
    }

	var builder strings.Builder

	// Add title
	if bc.Title != "" {
		builder.WriteString(bc.Title + "\n")
		builder.WriteString(strings.Repeat("=", len(bc.Title)) + "\n")
	}

    // Calculate the maximum duration for scaling
    maxDuration := time.Duration(0)
    for i := range results {
        if results[i].Mean.Duration > maxDuration {
            maxDuration = results[i].Mean.Duration
        }
    }

    if maxDuration == 0 {
        return msgNoValidDurations
    }

	// Calculate bar width (reserve space for labels and values)
	labelWidth := bc.calculateLabelWidth(results)
	valueWidth := 10                                           // Space for duration display
	barWidth := bc.Options.Width - labelWidth - valueWidth - 5 // 5 for spacing

	if barWidth < 10 {
		barWidth = 10 // Minimum bar width
	}

    // Render each result as a bar
    for i := range results {
        // Calculate bar length
        ratio := float64(results[i].Mean.Duration) / float64(maxDuration)
        barLength := int(ratio * float64(barWidth))

		// Create the bar
		bar := strings.Repeat("█", barLength)

		// Apply colors based on benchmark name
        if bc.Options.UseColors {
            bar = bc.colorizeBar(results[i].Name, bar)
        }

		// Format the line
        label := fmt.Sprintf("%-*s", labelWidth, results[i].Name)
        value := fmt.Sprintf("%8s", formatDuration(results[i].Mean.Duration))

		builder.WriteString(fmt.Sprintf("%s %s %s\n", label, bar, value))
	}

	// Add comparison if multiple results
	if len(results) > 1 {
		builder.WriteString("\n")
		builder.WriteString(bc.renderComparison(results))
	}

	return builder.String()
}

// SparklineChart creates compact trend visualizations
type SparklineChart struct {
	Options VisualizerOptions
	Title   string
}

// NewSparklineChart creates a new sparkline chart visualizer
func NewSparklineChart(title string) *SparklineChart {
	return &SparklineChart{
		Options: DefaultVisualizerOptions(),
		Title:   title,
	}
}

// Render creates a sparkline chart from benchmark results
func (sc *SparklineChart) Render(results []Result) string {
    if len(results) == 0 {
        return msgNoResults
    }

	var builder strings.Builder

	if sc.Title != "" {
		builder.WriteString(sc.Title + ": ")
	}

	// Convert durations to sparkline characters
	sparkline := sc.createSparkline(results)
	builder.WriteString(sparkline)

	if sc.Options.ShowValues {
		builder.WriteString(fmt.Sprintf(" (min: %s, max: %s)",
			formatDuration(sc.getMinDuration(results)),
			formatDuration(sc.getMaxDuration(results))))
	}

	return builder.String()
}

// ComparisonTable creates side-by-side comparison tables
type ComparisonTable struct {
	Options VisualizerOptions
	Title   string
}

// NewComparisonTable creates a new comparison table visualizer
func NewComparisonTable(title string) *ComparisonTable {
	return &ComparisonTable{
		Options: DefaultVisualizerOptions(),
		Title:   title,
	}
}

// Render creates a comparison table from benchmark results
func (ct *ComparisonTable) Render(results []Result) string {
	if len(results) == 0 {
		return "No results to display"
	}

	var builder strings.Builder

	if ct.Title != "" {
		builder.WriteString(ct.Title + "\n")
		builder.WriteString(strings.Repeat("=", len(ct.Title)) + "\n")
	}

	// Create table headers
	headers := []string{"Benchmark", "Mean", "Median", "Min", "Max", "StdDev", "Speedup"}
	colWidths := ct.calculateColumnWidths(results, headers)

	// Render header row
	builder.WriteString(ct.renderTableRow(headers, colWidths, true))
	builder.WriteString(ct.renderSeparator(colWidths))

	// Find baseline (fastest) for speedup calculation
    baseline := ct.findBaseline(results)

    // Render data rows
    for i := range results {
        speedup := ct.calculateSpeedup(baseline, &results[i])
        row := []string{
            results[i].Name,
            formatDuration(results[i].Mean.Duration),
            formatDuration(results[i].Median.Duration),
            formatDuration(results[i].Min.Duration),
            formatDuration(results[i].Max.Duration),
            formatDuration(results[i].StdDev.Duration),
            speedup,
        }
        builder.WriteString(ct.renderTableRow(row, colWidths, false))
    }

	return builder.String()
}

// Helper functions

func (bc *BarChart) calculateLabelWidth(results []Result) int {
	maxLen := 0
    for i := range results {
        if len(results[i].Name) > maxLen {
            maxLen = len(results[i].Name)
        }
    }
	return maxLen + 2 // Add padding
}

func (bc *BarChart) colorizeBar(name, bar string) string {
	if !bc.Options.UseColors {
		return bar
	}

	// Color mapping based on common benchmark names
	switch {
	case strings.Contains(strings.ToLower(name), "mitl"):
		return "\033[32m" + bar + "\033[0m" // Green
	case strings.Contains(strings.ToLower(name), "docker"):
		return "\033[34m" + bar + "\033[0m" // Blue
	case strings.Contains(strings.ToLower(name), "podman"):
		return "\033[33m" + bar + "\033[0m" // Yellow
	default:
		return bar
	}
}

func (bc *BarChart) renderComparison(results []Result) string {
    if len(results) < 2 {
        return ""
    }

	// Sort by duration to find fastest
	sorted := make([]Result, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Mean.Duration < sorted[j].Mean.Duration
	})

	fastest := sorted[0]
	var comparisons []string

    for i := 1; i < len(sorted); i++ {
        ratio := float64(sorted[i].Mean.Duration) / float64(fastest.Mean.Duration)
        comparisons = append(comparisons,
            fmt.Sprintf("%.1fx slower than %s", ratio, fastest.Name))
    }

	return fmt.Sprintf("Speed improvement: %s is %s\n",
		fastest.Name, strings.Join(comparisons, ", "))
}

func (sc *SparklineChart) createSparkline(results []Result) string {
	if len(results) == 0 {
		return ""
	}

	// Sparkline characters from lowest to highest
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Find min and max durations
	minDur := sc.getMinDuration(results)
	maxDur := sc.getMaxDuration(results)

	if minDur == maxDur {
		// Use ▄ for identical values (index 3)
		return strings.Repeat(string(chars[3]), len(results))
	}

	var sparkline strings.Builder
	durRange := float64(maxDur - minDur)

    for i := range results {
        // Normalize to 0-1 range
        normalized := float64(results[i].Mean.Duration-minDur) / durRange
		// Map to character index
		charIndex := int(normalized * float64(len(chars)-1))
		if charIndex >= len(chars) {
			charIndex = len(chars) - 1
		}
		sparkline.WriteRune(chars[charIndex])
	}

	return sparkline.String()
}

func (sc *SparklineChart) getMinDuration(results []Result) time.Duration {
	if len(results) == 0 {
		return 0
	}
	min := results[0].Mean.Duration
	for _, result := range results[1:] {
		if result.Mean.Duration < min {
			min = result.Mean.Duration
		}
	}
	return min
}

func (sc *SparklineChart) getMaxDuration(results []Result) time.Duration {
	if len(results) == 0 {
		return 0
	}
	max := results[0].Mean.Duration
	for _, result := range results[1:] {
		if result.Mean.Duration > max {
			max = result.Mean.Duration
		}
	}
	return max
}

func (ct *ComparisonTable) calculateColumnWidths(results []Result, headers []string) []int {
	widths := make([]int, len(headers))

	// Initialize with header lengths
	for i, header := range headers {
		widths[i] = len(header)
	}

    // Check data lengths
    for idx := range results {
        data := []string{
            results[idx].Name,
            formatDuration(results[idx].Mean.Duration),
            formatDuration(results[idx].Median.Duration),
            formatDuration(results[idx].Min.Duration),
            formatDuration(results[idx].Max.Duration),
            formatDuration(results[idx].StdDev.Duration),
            "1.00x", // Placeholder for speedup
        }

        for i, val := range data {
            if i < len(widths) && len(val) > widths[i] {
                widths[i] = len(val)
            }
        }
    }

	// Add padding
	for i := range widths {
		widths[i] += 2
	}

	return widths
}

func (ct *ComparisonTable) renderTableRow(columns []string, widths []int, isHeader bool) string {
	var row strings.Builder
	row.WriteString("|")

	for i, col := range columns {
		if i < len(widths) {
			format := fmt.Sprintf(" %%-%ds|", widths[i]-1)
			row.WriteString(fmt.Sprintf(format, col))
		}
	}
	row.WriteString("\n")

	return row.String()
}

func (ct *ComparisonTable) renderSeparator(widths []int) string {
	var sep strings.Builder
	sep.WriteString("|")

	for _, width := range widths {
		sep.WriteString(strings.Repeat("-", width) + "|")
	}
	sep.WriteString("\n")

	return sep.String()
}

func (ct *ComparisonTable) findBaseline(results []Result) *Result {
    if len(results) == 0 {
        return &Result{}
    }

    baseline := &results[0]
    for i := 1; i < len(results); i++ {
        if results[i].Mean.Duration < baseline.Mean.Duration {
            baseline = &results[i]
        }
    }
    return baseline
}

func (ct *ComparisonTable) calculateSpeedup(baseline *Result, result *Result) string {
    if baseline == nil || baseline.Mean.Duration == 0 {
        return "N/A"
    }

    ratio := float64(result.Mean.Duration) / float64(baseline.Mean.Duration)
    return fmt.Sprintf("%.2fx", ratio)
}

// Utility functions

func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fµs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1000000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func getTerminalWidth() int {
	// Try to get terminal width from environment or system calls
	if width := os.Getenv("COLUMNS"); width != "" {
		if w, err := strconv.Atoi(width); err == nil && w > 0 {
			return w
		}
	}

	// Default fallback
	return 80
}

// Helper function to determine if a value represents an error condition
func isErrorResult(result *Result) bool {
    if result == nil {
        return true
    }
    return !result.Success || result.Error != ""
}

// FormatResults provides a quick way to format results with a default bar chart
func FormatResults(results []Result, title string) string {
	chart := NewBarChart(title)
	return chart.Render(results)
}

// FormatTrend provides a quick way to format results as a sparkline
func FormatTrend(results []Result, title string) string {
	chart := NewSparklineChart(title)
	return chart.Render(results)
}

// FormatComparison provides a quick way to format results as a comparison table
func FormatComparison(results []Result, title string) string {
	table := NewComparisonTable(title)
	return table.Render(results)
}
