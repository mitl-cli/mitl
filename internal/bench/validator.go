package bench

import (
	"fmt"
	"strings"
	"time"
)

// ResultValidator defines the interface for validating benchmark results
type ResultValidator interface {
	Validate(result Result) error
}

// StatisticalValidator validates statistical properties of benchmark results
type StatisticalValidator struct {
	// MinSamples is the minimum number of samples required for statistical validity
	MinSamples int
	// MaxCoefficient is the maximum allowed coefficient of variation (stddev/mean)
	MaxCoefficient float64
}

// NewStatisticalValidator creates a new statistical validator with default settings
func NewStatisticalValidator() *StatisticalValidator {
	return &StatisticalValidator{
		MinSamples:     3,   // At least 3 samples for basic statistics
		MaxCoefficient: 0.5, // 50% coefficient of variation maximum
	}
}

// Validate checks if the result has valid statistical properties
func (sv *StatisticalValidator) Validate(result Result) error {
	// Convert durations to milliseconds for easier comparison
	meanMs := float64(result.Mean.Nanoseconds()) / 1e6
	minMs := float64(result.Min.Nanoseconds()) / 1e6
	maxMs := float64(result.Max.Nanoseconds()) / 1e6
	stdDevMs := float64(result.StdDev.Nanoseconds()) / 1e6
	p95Ms := float64(result.P95.Nanoseconds()) / 1e6
	p99Ms := float64(result.P99.Nanoseconds()) / 1e6

	// Check that Mean is between Min and Max
	if meanMs < minMs {
		return fmt.Errorf("mean (%.2f ms) is less than minimum (%.2f ms) for benchmark %s",
			meanMs, minMs, result.Name)
	}

	if meanMs > maxMs {
		return fmt.Errorf("mean (%.2f ms) is greater than maximum (%.2f ms) for benchmark %s",
			meanMs, maxMs, result.Name)
	}

	// Check that StdDev is non-negative
	if stdDevMs < 0 {
		return fmt.Errorf("standard deviation (%.2f ms) cannot be negative for benchmark %s",
			stdDevMs, result.Name)
	}

	// Check percentiles are in correct order
	if p95Ms > p99Ms {
		return fmt.Errorf("P95 (%.2f ms) cannot be greater than P99 (%.2f ms) for benchmark %s",
			p95Ms, p99Ms, result.Name)
	}

	// Check coefficient of variation if we have a mean > 0
	if meanMs > 0 {
		coefficient := stdDevMs / meanMs
		if coefficient > sv.MaxCoefficient {
			return fmt.Errorf("coefficient of variation (%.3f) exceeds maximum (%.3f) for benchmark %s",
				coefficient, sv.MaxCoefficient, result.Name)
		}
	}

	// Validate that percentiles are within Min-Max range
	if p95Ms < minMs || p95Ms > maxMs {
		return fmt.Errorf("P95 (%.2f ms) is outside Min-Max range [%.2f, %.2f] for benchmark %s",
			p95Ms, minMs, maxMs, result.Name)
	}

	if p99Ms < minMs || p99Ms > maxMs {
		return fmt.Errorf("P99 (%.2f ms) is outside Min-Max range [%.2f, %.2f] for benchmark %s",
			p99Ms, minMs, maxMs, result.Name)
	}

	// Check minimum sample size
	if result.Iterations < sv.MinSamples {
		return fmt.Errorf("insufficient samples (%d) for statistical validity, minimum required: %d for benchmark %s",
			result.Iterations, sv.MinSamples, result.Name)
	}

	return nil
}

// RangeValidator validates that values are within acceptable ranges
type RangeValidator struct {
	MinDuration time.Duration
	MaxDuration time.Duration
}

// NewRangeValidator creates a new range validator with specified duration limits
func NewRangeValidator(minDuration, maxDuration time.Duration) *RangeValidator {
	return &RangeValidator{
		MinDuration: minDuration,
		MaxDuration: maxDuration,
	}
}

// Validate checks if the result duration is within acceptable ranges
func (rv *RangeValidator) Validate(result Result) error {
	// Check total time against limits
	if result.TotalTime.Duration < rv.MinDuration {
		return fmt.Errorf("total duration (%v) is below minimum threshold (%v) for benchmark %s",
			result.TotalTime.Duration, rv.MinDuration, result.Name)
	}

	if result.TotalTime.Duration > rv.MaxDuration {
		return fmt.Errorf("total duration (%v) exceeds maximum threshold (%v) for benchmark %s",
			result.TotalTime.Duration, rv.MaxDuration, result.Name)
	}

	// Validate that statistical values are within reasonable bounds
	minMs := rv.MinDuration.Seconds() * 1000
	maxMs := rv.MaxDuration.Seconds() * 1000

	resultMinMs := float64(result.Min.Nanoseconds()) / 1e6
	resultMaxMs := float64(result.Max.Nanoseconds()) / 1e6

	if resultMinMs < minMs {
		return fmt.Errorf("minimum time (%.2f ms) is below threshold (%.2f ms) for benchmark %s",
			resultMinMs, minMs, result.Name)
	}

	if resultMaxMs > maxMs {
		return fmt.Errorf("maximum time (%.2f ms) exceeds threshold (%.2f ms) for benchmark %s",
			resultMaxMs, maxMs, result.Name)
	}

	return nil
}

// ConsistencyValidator validates result consistency across runs
type ConsistencyValidator struct {
	MaxVariance float64 // Maximum allowed variance in results
}

// NewConsistencyValidator creates a new consistency validator
func NewConsistencyValidator(maxVariance float64) *ConsistencyValidator {
	return &ConsistencyValidator{
		MaxVariance: maxVariance,
	}
}

// Validate checks if the result shows consistent performance
func (cv *ConsistencyValidator) Validate(result Result) error {
	minMs := float64(result.Min.Nanoseconds()) / 1e6
	maxMs := float64(result.Max.Nanoseconds()) / 1e6
	meanMs := float64(result.Mean.Nanoseconds()) / 1e6
	stdDevMs := float64(result.StdDev.Nanoseconds()) / 1e6

	// Check for extreme outliers
	if maxMs > 0 && minMs > 0 {
		ratio := maxMs / minMs
		if ratio > 10.0 { // Max is more than 10x the minimum
			return fmt.Errorf("extreme variance detected: max/min ratio (%.2f) indicates inconsistent results for benchmark %s",
				ratio, result.Name)
		}
	}

	// Check variance relative to mean
	if meanMs > 0 {
		variance := stdDevMs * stdDevMs
		relativeVariance := variance / meanMs * meanMs

		if relativeVariance > cv.MaxVariance {
			return fmt.Errorf("relative variance (%.3f) exceeds maximum (%.3f) for benchmark %s",
				relativeVariance, cv.MaxVariance, result.Name)
		}
	}

	// Check that the result was successful for consistency validation
	if !result.Success {
		return fmt.Errorf("benchmark %s failed, cannot validate consistency", result.Name)
	}

	return nil
}

// FormatValidator validates export format and data integrity
type FormatValidator struct {
	RequiredFields []string
}

// NewFormatValidator creates a new format validator
func NewFormatValidator() *FormatValidator {
	return &FormatValidator{
		RequiredFields: []string{"name", "duration", "success", "statistics"},
	}
}

// Validate checks if the result has proper format and required fields
func (fv *FormatValidator) Validate(result Result) error {
	// Check required string fields
	if strings.TrimSpace(result.Name) == "" {
		return fmt.Errorf("benchmark name cannot be empty")
	}

	// Check duration is valid (total time should be positive)
	if result.TotalTime.Duration < 0 {
		return fmt.Errorf("total duration cannot be negative for benchmark %s", result.Name)
	}

	// Check timestamps are valid
	if !result.Timestamp.IsZero() {
		// Basic timestamp validation - should be reasonable
		now := time.Now()
		if result.Timestamp.After(now.Add(time.Hour)) {
			return fmt.Errorf("timestamp appears to be in the future for benchmark %s", result.Name)
		}
		if result.Timestamp.Before(now.Add(-24 * time.Hour)) {
			return fmt.Errorf("timestamp appears to be too old (>24h) for benchmark %s", result.Name)
		}
	}

	// Check for required statistical fields - at least one duration should be set
	if result.Mean.Duration == 0 && result.Min.Duration == 0 && result.Max.Duration == 0 {
		return fmt.Errorf("statistics appear to be uninitialized for benchmark %s", result.Name)
	}

	// Validate iterations count
	if result.Iterations < 0 {
		return fmt.Errorf("iterations count cannot be negative for benchmark %s", result.Name)
	}

	// Check category is valid
	validCategories := map[Category]bool{
		CategoryBuild:  true,
		CategoryRun:    true,
		CategoryCache:  true,
		CategoryVolume: true,
		CategoryE2E:    true,
	}

	if result.Category != "" && !validCategories[result.Category] {
		return fmt.Errorf("invalid category '%s' for benchmark %s", result.Category, result.Name)
	}

	return nil
}

// ValidateResults validates a slice of results using multiple validators
func ValidateResults(results []Result) []error {
	var errors []error

	// Create default validators
	validators := []ResultValidator{
		NewStatisticalValidator(),
		NewRangeValidator(time.Microsecond, 30*time.Minute), // 1μs to 30min
		NewConsistencyValidator(0.25),                       // 25% max relative variance
		NewFormatValidator(),
	}

	// Validate each result with each validator
	for _, result := range results {
		for _, validator := range validators {
			if err := validator.Validate(result); err != nil {
				errors = append(errors, fmt.Errorf("validation failed for %s: %w", result.Name, err))
			}
		}
	}

	return errors
}

// ValidateBenchmarkExecution validates that a benchmark suite executed properly
func ValidateBenchmarkExecution(suite *Suite) error {
	if suite == nil {
		return fmt.Errorf("benchmark suite cannot be nil")
	}

	benchmarks := suite.GetBenchmarks()
	if len(benchmarks) == 0 {
		return fmt.Errorf("benchmark suite must contain at least one benchmark")
	}

	// Check that all benchmarks have valid names
	nameMap := make(map[string]bool)
	for _, benchmark := range benchmarks {
		if strings.TrimSpace(benchmark.Name) == "" {
			return fmt.Errorf("benchmark name cannot be empty")
		}

		// Check for duplicate names
		if nameMap[benchmark.Name] {
			return fmt.Errorf("duplicate benchmark name: %s", benchmark.Name)
		}
		nameMap[benchmark.Name] = true

		// Validate benchmark runner exists
		if benchmark.Runner == nil {
			return fmt.Errorf("benchmark runner cannot be nil for benchmark %s", benchmark.Name)
		}
	}

	// Validate suite configuration by accessing config through a getter method
	// Since config is private, we'll validate the results instead
	results := suite.Results()
	if len(results) > 0 {
		// If we have results, validate them
		validationErrors := ValidateResults(results)
		if len(validationErrors) > 0 {
			return fmt.Errorf("suite execution validation failed: %v", validationErrors[0])
		}
	}

	return nil
}

// validateBenchmarkConfig validates a single benchmark configuration
func validateBenchmarkConfig(config BenchmarkConfig) error {
	if strings.TrimSpace(config.Image) == "" {
		return fmt.Errorf("benchmark image cannot be empty")
	}

	// Validate environment variables don't contain invalid characters
	for key, value := range config.Environment {
		if strings.Contains(key, "=") {
			return fmt.Errorf("environment key cannot contain '=': %s", key)
		}
		if key == "" {
			return fmt.Errorf("environment key cannot be empty")
		}
		// Value can be empty, but check for null bytes which could cause issues
		if strings.Contains(value, "\x00") {
			return fmt.Errorf("environment value contains null byte for key %s", key)
		}
	}

	// Validate volume mounts format (basic check)
	for _, mount := range config.VolumeMounts {
		if !strings.Contains(mount, ":") {
			return fmt.Errorf("volume mount must contain ':' separator: %s", mount)
		}
		parts := strings.Split(mount, ":")
		if len(parts) < 2 {
			return fmt.Errorf("volume mount must have source and destination: %s", mount)
		}
		if parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("volume mount source and destination cannot be empty: %s", mount)
		}
	}

	return nil
}

// ValidateResultStatistically performs advanced statistical validation on a result
func ValidateResultStatistically(result Result) error {
	stats := result.ToStatistics()

	// Check for statistical impossibilities
	if stats.Mean < stats.Min || stats.Mean > stats.Max {
		return fmt.Errorf("mean (%.2f) must be between min (%.2f) and max (%.2f) for %s",
			stats.Mean, stats.Min, stats.Max, result.Name)
	}

	if stats.StdDev < 0 {
		return fmt.Errorf("standard deviation cannot be negative for %s", result.Name)
	}

	if stats.P95 < stats.Min || stats.P95 > stats.Max {
		return fmt.Errorf("P95 (%.2f) must be between min (%.2f) and max (%.2f) for %s",
			stats.P95, stats.Min, stats.Max, result.Name)
	}

	if stats.P99 < stats.P95 {
		return fmt.Errorf("P99 (%.2f) cannot be less than P95 (%.2f) for %s",
			stats.P99, stats.P95, result.Name)
	}

	if stats.P99 < stats.Min || stats.P99 > stats.Max {
		return fmt.Errorf("P99 (%.2f) must be between min (%.2f) and max (%.2f) for %s",
			stats.P99, stats.Min, stats.Max, result.Name)
	}

	return nil
}

// ValidateComparativeResults validates results from comparison tools
func ValidateComparativeResults(mitlResults, comparisonResults []Result, toolName string) error {
	if len(mitlResults) == 0 {
		return fmt.Errorf("mitl results cannot be empty")
	}

	if len(comparisonResults) == 0 {
		return fmt.Errorf("%s comparison results cannot be empty", toolName)
	}

	// Validate each comparison result
	for _, result := range comparisonResults {
		if err := ValidateResultStatistically(result); err != nil {
			return fmt.Errorf("%s result validation failed: %w", toolName, err)
		}

		// Check that the result name indicates it's from the comparison tool
		expectedSuffix := fmt.Sprintf("_%s", strings.ToLower(toolName))
		if !strings.HasSuffix(result.Name, expectedSuffix) {
			return fmt.Errorf("comparison result name '%s' should end with '%s'",
				result.Name, expectedSuffix)
		}
	}

	return nil
}
