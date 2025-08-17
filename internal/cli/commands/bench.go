package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"mitl/internal/bench"
)

// BenchCommand implements the bench command with subcommands: run, compare, list, export
type BenchCommand struct {
	iterations   int
	category     string
	compareWith  string
	output       string
	format       string
	verbose      bool
	parallel     bool
	showProgress bool
}

// NewBenchCommand creates a new bench command instance
func NewBenchCommand() *BenchCommand {
	return &BenchCommand{
		iterations:   10,
		category:     "",
		compareWith:  "",
		output:       "",
		format:       "table",
		verbose:      false,
		parallel:     false,
		showProgress: true,
	}
}

// Name returns the command name
func (bc *BenchCommand) Name() string {
	return "bench"
}

// Description returns the command description
func (bc *BenchCommand) Description() string {
	return "Run benchmarks and performance comparisons"
}

// Run executes the bench command with subcommands
func (bc *BenchCommand) Run(args []string) error {
	if len(args) == 0 {
		return bc.printUsage()
	}

	// Parse global flags first
	args, err := bc.parseGlobalFlags(args)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if len(args) == 0 {
		return bc.printUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "run":
		return bc.runBenchmarks(subArgs)
	case "compare":
		return bc.runComparison(subArgs)
	case "list":
		return bc.listBenchmarks(subArgs)
	case "export":
		return bc.exportResults(subArgs)
	case "help", "--help", "-h":
		return bc.printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s", subcommand)
	}
}

// runBenchmarks executes the benchmark suite
func (bc *BenchCommand) runBenchmarks(args []string) error {
	fmt.Println("🚀 Starting mitl benchmarks...")
	fmt.Printf("Configuration: iterations=%d, category=%s, parallel=%v\n",
		bc.iterations, bc.getFilterDescription(), bc.parallel)
	fmt.Println()

	// Create benchmark suite with configuration
	config := bench.Config{
		MinIterations:     bc.iterations,
		MaxIterations:     bc.iterations,
		Duration:          30 * time.Second,
		WarmupIterations:  2,
		CooldownDuration:  100 * time.Millisecond,
		CollectMemoryInfo: true,
		Parallel:          bc.parallel,
		Verbose:           bc.verbose,
	}

	suite := bench.NewSuite(config)

	// Register benchmarks based on category filter
	if err := bc.registerBenchmarks(suite); err != nil {
		return fmt.Errorf("failed to register benchmarks: %w", err)
	}

	benchmarks := suite.GetBenchmarks()
	if len(benchmarks) == 0 {
		fmt.Println("⚠️  No benchmarks match the specified criteria")
		return nil
	}

	fmt.Printf("📊 Running %d benchmark(s)...\n\n", len(benchmarks))

	// Run benchmarks with progress indication
	var results []bench.Result
	var err error
	if bc.showProgress {
		results, err = bc.runWithProgress(suite, benchmarks)
	} else {
		results, err = suite.Run()
	}

	if err != nil {
		return fmt.Errorf("benchmark execution failed: %w", err)
	}

	// Display results
	bc.displayResults(results)

	// Save results if output specified
	if bc.output != "" {
		if err := bc.saveResults(results); err != nil {
			fmt.Printf("⚠️  Failed to save results: %v\n", err)
		} else {
			fmt.Printf("📁 Results saved to: %s\n", bc.output)
		}
	}

	return nil
}

// runComparison executes benchmarks and compares with Docker/Podman
func (bc *BenchCommand) runComparison(args []string) error {
	if bc.compareWith == "" {
		bc.compareWith = "docker" // Default comparison
	}

	fmt.Printf("🔍 Running comparison benchmarks (mitl vs %s)...\n", bc.compareWith)
	fmt.Println()

	// Create and run mitl benchmarks first
	config := bench.DefaultConfig()
	config.MinIterations = bc.iterations
	config.MaxIterations = bc.iterations
	config.Verbose = bc.verbose

	suite := bench.NewSuite(config)
	if err := bc.registerBenchmarks(suite); err != nil {
		return fmt.Errorf("failed to register benchmarks: %w", err)
	}

	benchmarks := suite.GetBenchmarks()
	if len(benchmarks) == 0 {
		fmt.Println("⚠️  No benchmarks available for comparison")
		return nil
	}

	fmt.Printf("📊 Running mitl benchmarks (%d)...\n", len(benchmarks))
	mitlResults, err := bc.runWithProgress(suite, benchmarks)
	if err != nil {
		return fmt.Errorf("mitl benchmark execution failed: %w", err)
	}

	// Run comparison benchmarks
	fmt.Printf("\n📊 Running %s benchmarks...\n", bc.compareWith)
	var comparisonResults []bench.Result

	switch bc.compareWith {
	case "docker":
		comparisonResults, err = bench.RunDockerComparison(benchmarks)
	case "podman":
		comparisonResults, err = bench.RunPodmanComparison(benchmarks)
	default:
		return fmt.Errorf("unsupported comparison tool: %s (supported: docker, podman)", bc.compareWith)
	}

	if err != nil {
		fmt.Printf("⚠️  %s comparison failed: %v\n", bc.compareWith, err)
		fmt.Println("Showing mitl results only...")
		bc.displayResults(mitlResults)
		return nil
	}

	// Generate comparison report
	report := bench.NewComparisonReport(mitlResults)
	switch bc.compareWith {
	case "docker":
		report.AddDockerResults(comparisonResults)
	case "podman":
		report.AddPodmanResults(comparisonResults)
	}

	fmt.Println("\n" + report.Generate())

	// Save comparison results if output specified
	if bc.output != "" {
		resultSets := map[string][]bench.Result{
			"mitl":         mitlResults,
			bc.compareWith: comparisonResults,
		}

		if err := bench.ExportComparison(resultSets, bc.output, bc.format); err != nil {
			fmt.Printf("⚠️  Failed to save comparison results: %v\n", err)
		} else {
			fmt.Printf("📁 Comparison results saved to: %s\n", bc.output)
		}
	}

	return nil
}

// listBenchmarks displays available benchmarks
func (bc *BenchCommand) listBenchmarks(args []string) error {
	fmt.Println("📋 Available Benchmarks")
	fmt.Println("=====================")
	fmt.Println()

	categories := map[bench.Category][]string{
		bench.CategoryBuild: {
			"build_simple_node      - Simple Node.js application build",
			"build_multi_stage      - Multi-stage Dockerfile build",
			"build_large_dependencies - Large dependency installation",
		},
		bench.CategoryRun: {
			"run_startup_time       - Container startup latency",
			"run_command_execution  - Command execution performance",
			"run_interactive        - Interactive container performance",
		},
		bench.CategoryCache: {
			"cache_cold_scenario    - Cold cache performance",
			"cache_warm_scenario    - Warm cache performance",
		},
		bench.CategoryVolume: {
			"volume_mount           - Volume mount/unmount performance",
			"volume_read            - Volume read I/O performance",
			"volume_write           - Volume write I/O performance",
			"volume_copy            - Volume copy operations",
		},
	}

	for category, benchmarks := range categories {
		fmt.Printf("🏷️  %s\n", strings.ToUpper(string(category)))
		for _, desc := range benchmarks {
			fmt.Printf("   %s\n", desc)
		}
		fmt.Println()
	}

	fmt.Println("Usage Examples:")
	fmt.Println("   mitl bench run                     # Run all benchmarks")
	fmt.Println("   mitl bench run --category=build    # Run only build benchmarks")
	fmt.Println("   mitl bench run --iterations=20     # Run with 20 iterations")
	fmt.Println("   mitl bench compare --with=docker   # Compare with Docker")
	fmt.Println("   mitl bench export --format=json    # Export previous results")

	return nil
}

// exportResults exports previous benchmark results
func (bc *BenchCommand) exportResults(args []string) error {
	// Parse export-specific arguments
	if err := bc.parseExportArgs(args); err != nil {
		return fmt.Errorf("failed to parse export arguments: %w", err)
	}

	if bc.output == "" {
		return fmt.Errorf("output file must be specified with --output flag")
	}

	// For now, return an informative message since we don't have persistent storage
	fmt.Println("📤 Export Results")
	fmt.Println("=================")
	fmt.Println()
	fmt.Printf("To export results, run benchmarks with --output flag:\n")
	fmt.Printf("   mitl bench run --output=%s --format=%s\n", bc.output, bc.format)
	fmt.Println()
	fmt.Printf("Supported formats: json, csv, markdown, html\n")

	return nil
}

// registerBenchmarks registers benchmarks based on category filter
func (bc *BenchCommand) registerBenchmarks(suite *bench.Suite) error {
	categories := bc.getCategoriesToRun()

	for _, category := range categories {
		switch category {
		case bench.CategoryBuild:
			if err := bc.registerBuildBenchmarks(suite); err != nil {
				return fmt.Errorf("failed to register build benchmarks: %w", err)
			}
		case bench.CategoryRun:
			if err := bc.registerRunBenchmarks(suite); err != nil {
				return fmt.Errorf("failed to register run benchmarks: %w", err)
			}
		case bench.CategoryCache:
			if err := bc.registerCacheBenchmarks(suite); err != nil {
				return fmt.Errorf("failed to register cache benchmarks: %w", err)
			}
		case bench.CategoryVolume:
			if err := bc.registerVolumeBenchmarks(suite); err != nil {
				return fmt.Errorf("failed to register volume benchmarks: %w", err)
			}
		}
	}

	return nil
}

// registerBuildBenchmarks registers build-related benchmarks
func (bc *BenchCommand) registerBuildBenchmarks(suite *bench.Suite) error {
	benchmarks := []struct {
		name        string
		description string
		runner      bench.BenchmarkRunner
	}{
		{
			"build_simple_node",
			"Simple Node.js application build",
			bench.NewSimpleBuildBenchmark(bc.iterations),
		},
		{
			"build_multi_stage",
			"Multi-stage Dockerfile build",
			bench.NewMultiStageBuildBenchmark(bc.iterations),
		},
		{
			"build_large_dependencies",
			"Large dependency installation build",
			bench.NewLargeDependencyBuildBenchmark(bc.iterations),
		},
	}

	for _, b := range benchmarks {
		if err := suite.Register(b.name, b.description, bench.CategoryBuild, b.runner); err != nil {
			return fmt.Errorf("failed to register %s: %w", b.name, err)
		}
	}

	return nil
}

// registerRunBenchmarks registers runtime-related benchmarks
func (bc *BenchCommand) registerRunBenchmarks(suite *bench.Suite) error {
	benchmarks := []struct {
		name        string
		description string
		runner      bench.BenchmarkRunner
	}{
		{
			"run_startup_time",
			"Container startup latency measurement",
			bench.NewStartupTimeBenchmark(bc.iterations),
		},
		{
			"run_command_execution",
			"Command execution performance",
			bench.NewCommandExecutionBenchmark(bc.iterations),
		},
		{
			"run_interactive",
			"Interactive container performance",
			bench.NewInteractiveRunBenchmark(bc.iterations),
		},
	}

	for _, b := range benchmarks {
		if err := suite.Register(b.name, b.description, bench.CategoryRun, b.runner); err != nil {
			return fmt.Errorf("failed to register %s: %w", b.name, err)
		}
	}

	return nil
}

// registerCacheBenchmarks registers cache-related benchmarks
func (bc *BenchCommand) registerCacheBenchmarks(suite *bench.Suite) error {
	benchmarks := []struct {
		name        string
		description string
		runner      bench.BenchmarkRunner
	}{
		{
			"cache_cold_scenario",
			"Cold cache performance (no existing cache)",
			bench.NewColdCacheBenchmark(bc.iterations),
		},
		{
			"cache_warm_scenario",
			"Warm cache performance (existing cache)",
			bench.NewWarmCacheBenchmark(bc.iterations),
		},
	}

	for _, b := range benchmarks {
		if err := suite.Register(b.name, b.description, bench.CategoryCache, b.runner); err != nil {
			return fmt.Errorf("failed to register %s: %w", b.name, err)
		}
	}

	return nil
}

// registerVolumeBenchmarks registers volume-related benchmarks
func (bc *BenchCommand) registerVolumeBenchmarks(suite *bench.Suite) error {
	benchmarks := []struct {
		name        string
		description string
		runner      bench.BenchmarkRunner
	}{
		{
			"volume_mount",
			"Volume mount/unmount performance",
			bench.NewVolumeMountBenchmark(bc.iterations),
		},
		{
			"volume_io",
			"Volume I/O performance (read/write)",
			bench.NewVolumeIOBenchmark(bc.iterations),
		},
		{
			"volume_copy",
			"Volume copy operations",
			bench.NewVolumeCopyBenchmark(bc.iterations),
		},
	}

	for _, b := range benchmarks {
		if err := suite.Register(b.name, b.description, bench.CategoryVolume, b.runner); err != nil {
			return fmt.Errorf("failed to register %s: %w", b.name, err)
		}
	}

	return nil
}

// runWithProgress runs benchmarks with progress indication
func (bc *BenchCommand) runWithProgress(suite *bench.Suite, benchmarks []bench.Benchmark) ([]bench.Result, error) {
	results := make([]bench.Result, 0, len(benchmarks))

	for i, benchmark := range benchmarks {
		fmt.Printf("[%d/%d] Running %s...", i+1, len(benchmarks), benchmark.Name)

		// Run single benchmark
		singleSuite := bench.NewSuite(bench.Config{
			MinIterations:     bc.iterations,
			MaxIterations:     bc.iterations,
			CollectMemoryInfo: true,
			Verbose:           bc.verbose,
		})

		if err := singleSuite.Register(benchmark.Name, benchmark.Description, benchmark.Category, benchmark.Runner); err != nil {
			fmt.Printf(" ❌ Failed to register\n")
			continue
		}

		benchResults, err := singleSuite.Run()
		if err != nil {
			fmt.Printf(" ❌ Failed: %v\n", err)
			continue
		}

		if len(benchResults) > 0 {
			result := benchResults[0]
			if result.Success {
				fmt.Printf(" ✅ %s (avg: %s)\n",
					result.Name,
					bench.Duration{Duration: result.Mean.Duration}.Duration.String())
			} else {
				fmt.Printf(" ❌ Failed: %s\n", result.Error)
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// displayResults shows benchmark results in formatted tables and charts
func (bc *BenchCommand) displayResults(results []bench.Result) {
	if len(results) == 0 {
		fmt.Println("⚠️  No results to display")
		return
	}

	fmt.Println("\n📈 Benchmark Results")
	fmt.Println("==================")

	// Display formatted table
	fmt.Println(bench.FormatComparison(results, "Performance Summary"))

	// Display bar chart for visual comparison
	if len(results) > 1 {
		fmt.Println(bench.FormatResults(results, "Performance Comparison"))
	}

	// Display summary statistics
	bc.displaySummary(results)
}

// displaySummary shows summary statistics
func (bc *BenchCommand) displaySummary(results []bench.Result) {
	successful := 0
	var totalDuration time.Duration

	for _, result := range results {
		if result.Success {
			successful++
			totalDuration += result.Mean.Duration
		}
	}

	fmt.Println("\n📊 Summary")
	fmt.Println("----------")
	fmt.Printf("Total benchmarks:   %d\n", len(results))
	fmt.Printf("Successful:         %d\n", successful)
	fmt.Printf("Failed:             %d\n", len(results)-successful)

	if successful > 0 {
		avgDuration := totalDuration / time.Duration(successful)
		fmt.Printf("Average duration:   %s\n", avgDuration.String())
	}
}

// saveResults saves results to file in specified format
func (bc *BenchCommand) saveResults(results []bench.Result) error {
	return bench.ExportToFormat(results, bc.output, bc.format)
}

// getCategoriesToRun returns categories to run based on filter
func (bc *BenchCommand) getCategoriesToRun() []bench.Category {
	if bc.category == "" {
		// Run all categories
		return []bench.Category{
			bench.CategoryBuild,
			bench.CategoryRun,
			bench.CategoryCache,
			bench.CategoryVolume,
		}
	}

	switch bc.category {
	case "build":
		return []bench.Category{bench.CategoryBuild}
	case "run":
		return []bench.Category{bench.CategoryRun}
	case "cache":
		return []bench.Category{bench.CategoryCache}
	case "volume":
		return []bench.Category{bench.CategoryVolume}
	default:
		// Invalid category, run all
		return []bench.Category{
			bench.CategoryBuild,
			bench.CategoryRun,
			bench.CategoryCache,
			bench.CategoryVolume,
		}
	}
}

// getFilterDescription returns a description of the current filter
func (bc *BenchCommand) getFilterDescription() string {
	if bc.category == "" {
		return "all"
	}
	return bc.category
}

// parseGlobalFlags parses flags that apply to all subcommands
func (bc *BenchCommand) parseGlobalFlags(args []string) ([]string, error) {
	var remaining []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "--iterations=") {
			val := strings.TrimPrefix(arg, "--iterations=")
			iterations, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid iterations value: %s", val)
			}
			if iterations <= 0 {
				return nil, fmt.Errorf("iterations must be positive, got: %d", iterations)
			}
			bc.iterations = iterations
		} else if strings.HasPrefix(arg, "--category=") {
			bc.category = strings.TrimPrefix(arg, "--category=")
		} else if strings.HasPrefix(arg, "--output=") {
			bc.output = strings.TrimPrefix(arg, "--output=")
		} else if strings.HasPrefix(arg, "--format=") {
			bc.format = strings.TrimPrefix(arg, "--format=")
		} else if strings.HasPrefix(arg, "--compare-with=") || strings.HasPrefix(arg, "--with=") {
			if strings.HasPrefix(arg, "--compare-with=") {
				bc.compareWith = strings.TrimPrefix(arg, "--compare-with=")
			} else {
				bc.compareWith = strings.TrimPrefix(arg, "--with=")
			}
		} else if arg == "--verbose" || arg == "-v" {
			bc.verbose = true
		} else if arg == "--parallel" {
			bc.parallel = true
		} else if arg == "--no-progress" {
			bc.showProgress = false
		} else {
			remaining = append(remaining, arg)
		}
	}

	return remaining, nil
}

// parseExportArgs parses arguments specific to export subcommand
func (bc *BenchCommand) parseExportArgs(args []string) error {
	// Export args are already parsed by parseGlobalFlags
	// Validate format
	validFormats := map[string]bool{
		"json":     true,
		"csv":      true,
		"markdown": true,
		"md":       true,
		"html":     true,
		"table":    true,
	}

	if !validFormats[bc.format] {
		return fmt.Errorf("unsupported format: %s (supported: json, csv, markdown, html)", bc.format)
	}

	return nil
}

// printUsage displays command usage information
func (bc *BenchCommand) printUsage() error {
	fmt.Println("Usage: mitl bench <subcommand> [options]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  run        Run benchmarks")
	fmt.Println("  compare    Compare mitl performance with Docker/Podman")
	fmt.Println("  list       List available benchmarks")
	fmt.Println("  export     Export previous results")
	fmt.Println("  help       Show this help")
	fmt.Println()
	fmt.Println("Global Options:")
	fmt.Println("  --iterations=N      Number of iterations per benchmark (default: 10)")
	fmt.Println("  --category=TYPE     Filter by category: build, run, cache, volume")
	fmt.Println("  --output=FILE       Save results to file")
	fmt.Println("  --format=FORMAT     Output format: json, csv, markdown, html (default: table)")
	fmt.Println("  --verbose, -v       Enable verbose output")
	fmt.Println("  --parallel          Run benchmarks in parallel")
	fmt.Println("  --no-progress       Disable progress indicators")
	fmt.Println()
	fmt.Println("Compare Options:")
	fmt.Println("  --with=TOOL         Compare with: docker, podman (default: docker)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mitl bench run")
	fmt.Println("  mitl bench run --category=build --iterations=20")
	fmt.Println("  mitl bench compare --with=docker")
	fmt.Println("  mitl bench run --output=results.json --format=json")
	fmt.Println("  mitl bench list")

	return nil
}
