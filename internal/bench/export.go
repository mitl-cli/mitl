package bench

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Exporter provides an interface for exporting benchmark results to various formats
type Exporter interface {
	// Export writes benchmark results to the specified path
	Export(results []Result, path string) error
}

// ExportOptions configure the export behavior
type ExportOptions struct {
	IncludeSystemInfo bool // Include system information in exports
	PrettyPrint       bool // Format JSON with indentation
	IncludeMemory     bool // Include memory statistics
	Timestamp         bool // Add timestamp to exports
}

// DefaultExportOptions returns sensible defaults for exports
func DefaultExportOptions() ExportOptions {
	return ExportOptions{
		IncludeSystemInfo: true,
		PrettyPrint:       true,
		IncludeMemory:     true,
		Timestamp:         true,
	}
}

// ExportMetadata contains metadata about the export
type ExportMetadata struct {
	Timestamp    time.Time  `json:"timestamp"`
	Version      string     `json:"version,omitempty"`
	SystemInfo   SystemInfo `json:"system_info,omitempty"`
	ResultCount  int        `json:"result_count"`
	ExportFormat string     `json:"export_format"`
}

// SystemInfo contains system information for exports
type SystemInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	CPUs         int    `json:"cpus"`
	GoVersion    string `json:"go_version"`
}

// ExportDocument wraps results with metadata for structured exports
type ExportDocument struct {
	Metadata ExportMetadata `json:"metadata"`
	Results  []Result       `json:"results"`
}

// JSONExporter exports benchmark results to JSON format
type JSONExporter struct {
	Options ExportOptions
}

// NewJSONExporter creates a new JSON exporter
func NewJSONExporter() *JSONExporter {
	return &JSONExporter{
		Options: DefaultExportOptions(),
	}
}

// Export writes benchmark results to a JSON file
func (je *JSONExporter) Export(results []Result, path string) error {
	if err := ensureDir(path); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	// Create export document with metadata
	doc := je.createExportDocument(results)

	encoder := json.NewEncoder(file)
	if je.Options.PrettyPrint {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(doc); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// CSVExporter exports benchmark results to CSV format
type CSVExporter struct {
	Options ExportOptions
}

// NewCSVExporter creates a new CSV exporter
func NewCSVExporter() *CSVExporter {
	return &CSVExporter{
		Options: DefaultExportOptions(),
	}
}

// Export writes benchmark results to a CSV file
func (ce *CSVExporter) Export(results []Result, path string) error {
	if err := ensureDir(path); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers
	headers := ce.getCSVHeaders()
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Write data rows
    for i := range results {
        row := ce.resultToCSVRow(&results[i])
        if err := writer.Write(row); err != nil {
            return fmt.Errorf("failed to write CSV row: %w", err)
        }
    }

	return nil
}

// MarkdownExporter exports benchmark results to Markdown format
type MarkdownExporter struct {
	Options ExportOptions
}

// NewMarkdownExporter creates a new Markdown exporter
func NewMarkdownExporter() *MarkdownExporter {
	return &MarkdownExporter{
		Options: DefaultExportOptions(),
	}
}

// Export writes benchmark results to a Markdown file
func (me *MarkdownExporter) Export(results []Result, path string) error {
	if err := ensureDir(path); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	if err := me.writeMarkdownContent(file, results); err != nil {
		return fmt.Errorf("failed to write Markdown content: %w", err)
	}

	return nil
}

// HTMLExporter exports benchmark results to HTML format
type HTMLExporter struct {
	Options ExportOptions
}

// NewHTMLExporter creates a new HTML exporter
func NewHTMLExporter() *HTMLExporter {
	return &HTMLExporter{
		Options: DefaultExportOptions(),
	}
}

// Export writes benchmark results to an HTML file
func (he *HTMLExporter) Export(results []Result, path string) error {
	if err := ensureDir(path); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	if err := he.writeHTMLContent(file, results); err != nil {
		return fmt.Errorf("failed to write HTML content: %w", err)
	}

	return nil
}

// Implementation methods for JSONExporter

func (je *JSONExporter) createExportDocument(results []Result) ExportDocument {
	metadata := ExportMetadata{
		Timestamp:    time.Now(),
		ResultCount:  len(results),
		ExportFormat: "json",
	}

	if je.Options.IncludeSystemInfo {
		metadata.SystemInfo = getSystemInfo()
	}

	return ExportDocument{
		Metadata: metadata,
		Results:  results,
	}
}

// Implementation methods for CSVExporter

func (ce *CSVExporter) getCSVHeaders() []string {
	headers := []string{
		"name",
		"category",
		"description",
		"iterations",
		"mean_ns",
		"median_ns",
		"min_ns",
		"max_ns",
		"std_dev_ns",
		"p95_ns",
		"p99_ns",
		"total_time_ns",
		"success",
		"error",
		"timestamp",
	}

	if ce.Options.IncludeMemory {
		headers = append(headers,
			"alloc_bytes",
			"total_alloc_bytes",
			"sys_bytes",
			"num_gc",
			"heap_alloc_bytes",
			"heap_sys_bytes",
			"heap_idle_bytes",
			"heap_inuse_bytes",
		)
	}

	return headers
}

func (ce *CSVExporter) resultToCSVRow(result *Result) []string {
	row := []string{
		result.Name,
		string(result.Category),
		result.Description,
		strconv.Itoa(result.Iterations),
		strconv.FormatInt(result.Mean.Nanoseconds(), 10),
		strconv.FormatInt(result.Median.Nanoseconds(), 10),
		strconv.FormatInt(result.Min.Nanoseconds(), 10),
		strconv.FormatInt(result.Max.Nanoseconds(), 10),
		strconv.FormatInt(result.StdDev.Nanoseconds(), 10),
		strconv.FormatInt(result.P95.Nanoseconds(), 10),
		strconv.FormatInt(result.P99.Nanoseconds(), 10),
		strconv.FormatInt(result.TotalTime.Nanoseconds(), 10),
		strconv.FormatBool(result.Success),
		result.Error,
		result.Timestamp.Format(time.RFC3339),
	}

	if ce.Options.IncludeMemory {
            row = append(row,
                strconv.FormatUint(result.Memory.AllocBytes, 10),
                strconv.FormatUint(result.Memory.TotalAllocBytes, 10),
                strconv.FormatUint(result.Memory.SysBytes, 10),
                strconv.FormatUint(uint64(result.Memory.NumGC), 10),
                strconv.FormatUint(result.Memory.HeapAllocBytes, 10),
                strconv.FormatUint(result.Memory.HeapSysBytes, 10),
                strconv.FormatUint(result.Memory.HeapIdleBytes, 10),
                strconv.FormatUint(result.Memory.HeapInuseBytes, 10),
            )
        }

        return row
    }

// Implementation methods for MarkdownExporter

func (me *MarkdownExporter) writeMarkdownContent(file io.Writer, results []Result) error {
	// Write header
	fmt.Fprintf(file, "# Benchmark Results\n\n")

	if me.Options.Timestamp {
		fmt.Fprintf(file, "Generated at: %s\n\n", time.Now().Format(time.RFC3339))
	}

	if me.Options.IncludeSystemInfo {
		sysInfo := getSystemInfo()
		fmt.Fprintf(file, "## System Information\n\n")
		fmt.Fprintf(file, "- **OS**: %s\n", sysInfo.OS)
		fmt.Fprintf(file, "- **Architecture**: %s\n", sysInfo.Architecture)
		fmt.Fprintf(file, "- **CPUs**: %d\n", sysInfo.CPUs)
		fmt.Fprintf(file, "- **Go Version**: %s\n\n", sysInfo.GoVersion)
	}

	// Write results table
	fmt.Fprintf(file, "## Results\n\n")
	fmt.Fprintf(file, "| Benchmark | Category | Mean | Median | Min | Max | StdDev | Iterations | Success |\n")
	fmt.Fprintf(file, "|-----------|----------|------|--------|-----|-----|--------|------------|----------|\n")

    for i := range results {
        fmt.Fprintf(file, "| %s | %s | %s | %s | %s | %s | %s | %d | %v |\n",
            results[i].Name,
            results[i].Category,
            formatDuration(results[i].Mean.Duration),
            formatDuration(results[i].Median.Duration),
            formatDuration(results[i].Min.Duration),
            formatDuration(results[i].Max.Duration),
            formatDuration(results[i].StdDev.Duration),
            results[i].Iterations,
            results[i].Success,
        )
    }

	// Add comparison section if multiple results
	if len(results) > 1 {
		fmt.Fprintf(file, "\n## Performance Comparison\n\n")
		me.writeMarkdownComparison(file, results)
	}

	// Add memory information if enabled
	if me.Options.IncludeMemory {
		fmt.Fprintf(file, "\n## Memory Usage\n\n")
		me.writeMarkdownMemory(file, results)
	}

	return nil
}

func (me *MarkdownExporter) writeMarkdownComparison(file io.Writer, results []Result) {
	// Create a simple bar chart visualization
	chart := NewBarChart("Performance Comparison")
	chartOutput := chart.Render(results)

	fmt.Fprintf(file, "```\n%s```\n", chartOutput)
}

func (me *MarkdownExporter) writeMarkdownMemory(file io.Writer, results []Result) {
	fmt.Fprintf(file, "| Benchmark | Alloc | Total Alloc | Sys | Heap Alloc | Heap Sys |\n")
	fmt.Fprintf(file, "|-----------|-------|-------------|-----|------------|----------|\n")

    for i := range results {
        fmt.Fprintf(file, "| %s | %s | %s | %s | %s | %s |\n",
            results[i].Name,
            formatBytes(results[i].Memory.AllocBytes),
            formatBytes(results[i].Memory.TotalAllocBytes),
            formatBytes(results[i].Memory.SysBytes),
            formatBytes(results[i].Memory.HeapAllocBytes),
            formatBytes(results[i].Memory.HeapSysBytes),
        )
    }
}

// Implementation methods for HTMLExporter

func (he *HTMLExporter) writeHTMLContent(file io.Writer, results []Result) error {
	tmpl := template.Must(template.New("benchmark").Parse(htmlTemplate))

	data := struct {
		Timestamp  string
		SystemInfo SystemInfo
		Results    []Result
		Options    ExportOptions
		ChartData  string
	}{
		Timestamp:  time.Now().Format(time.RFC3339),
		SystemInfo: getSystemInfo(),
		Results:    results,
		Options:    he.Options,
		ChartData:  he.generateChartData(results),
	}

	return tmpl.Execute(file, data)
}

func (he *HTMLExporter) generateChartData(results []Result) string {
	// Generate JavaScript data for charts
	var names, values []string

    for i := range results {
        names = append(names, fmt.Sprintf("'%s'", results[i].Name))
        values = append(values, fmt.Sprintf("%.2f", results[i].Mean.Seconds()*1000)) // Convert to milliseconds
    }

	return fmt.Sprintf("{labels: [%s], data: [%s]}",
		strings.Join(names, ","),
		strings.Join(values, ","))
}

// Utility functions

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}

func getSystemInfo() SystemInfo {
	return SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		CPUs:         runtime.NumCPU(),
		GoVersion:    runtime.Version(),
	}
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ExportToFormat is a convenience function to export results in the specified format
func ExportToFormat(results []Result, path, format string) error {
	var exporter Exporter

	switch strings.ToLower(format) {
	case "json":
		exporter = NewJSONExporter()
	case "csv":
		exporter = NewCSVExporter()
	case "markdown", "md":
		exporter = NewMarkdownExporter()
	case "html":
		exporter = NewHTMLExporter()
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}

	return exporter.Export(results, path)
}

// ExportComparison exports a comparison report across multiple result sets
func ExportComparison(resultSets map[string][]Result, path, format string) error {
	// Flatten all results into a single slice with prefixed names
	var allResults []Result

    for setName, set := range resultSets {
        for i := range set {
            // Create a copy with prefixed name
            prefixedResult := set[i]
            prefixedResult.Name = fmt.Sprintf("%s_%s", setName, set[i].Name)
            allResults = append(allResults, prefixedResult)
        }
    }

	return ExportToFormat(allResults, path, format)
}

// HTML template for HTML exports
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Benchmark Results</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        table { border-collapse: collapse; width: 100%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .chart-container { width: 100%; height: 400px; margin: 20px 0; }
        .system-info { background-color: #f9f9f9; padding: 15px; border-radius: 5px; margin: 20px 0; }
    </style>
</head>
<body>
    <h1>Benchmark Results</h1>
    
    {{if .Options.Timestamp}}
    <p><strong>Generated:</strong> {{.Timestamp}}</p>
    {{end}}
    
    {{if .Options.IncludeSystemInfo}}
    <div class="system-info">
        <h2>System Information</h2>
        <p><strong>OS:</strong> {{.SystemInfo.OS}}</p>
        <p><strong>Architecture:</strong> {{.SystemInfo.Architecture}}</p>
        <p><strong>CPUs:</strong> {{.SystemInfo.CPUs}}</p>
        <p><strong>Go Version:</strong> {{.SystemInfo.GoVersion}}</p>
    </div>
    {{end}}
    
    <div class="chart-container">
        <canvas id="benchmarkChart"></canvas>
    </div>
    
    <table>
        <thead>
            <tr>
                <th>Benchmark</th>
                <th>Category</th>
                <th>Mean</th>
                <th>Median</th>
                <th>Min</th>
                <th>Max</th>
                <th>StdDev</th>
                <th>Iterations</th>
                <th>Success</th>
            </tr>
        </thead>
        <tbody>
            {{range .Results}}
            <tr>
                <td>{{.Name}}</td>
                <td>{{.Category}}</td>
                <td>{{.Mean.Duration}}</td>
                <td>{{.Median.Duration}}</td>
                <td>{{.Min.Duration}}</td>
                <td>{{.Max.Duration}}</td>
                <td>{{.StdDev.Duration}}</td>
                <td>{{.Iterations}}</td>
                <td>{{if .Success}}✅{{else}}❌{{end}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    
    <script>
        const ctx = document.getElementById('benchmarkChart').getContext('2d');
        const chartData = {{.ChartData}};
        
        new Chart(ctx, {
            type: 'bar',
            data: {
                labels: chartData.labels,
                datasets: [{
                    label: 'Mean Duration (ms)',
                    data: chartData.data,
                    backgroundColor: 'rgba(54, 162, 235, 0.2)',
                    borderColor: 'rgba(54, 162, 235, 1)',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Duration (ms)'
                        }
                    }
                }
            }
        });
    </script>
</body>
</html>`
