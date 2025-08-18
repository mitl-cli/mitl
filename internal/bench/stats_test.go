package bench

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// TestCalculateMean tests the CalculateMean function with various data sets
func TestCalculateMean(t *testing.T) {
	tests := []struct {
		name     string
		values   []time.Duration
		expected time.Duration
	}{
		{
			name:     "empty slice",
			values:   []time.Duration{},
			expected: 0,
		},
		{
			name:     "single value",
			values:   []time.Duration{100 * time.Millisecond},
			expected: 100 * time.Millisecond,
		},
		{
			name:     "simple average",
			values:   []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			expected: 150 * time.Millisecond,
		},
		{
			name:     "three values",
			values:   []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond},
			expected: 200 * time.Millisecond,
		},
		{
			name:     "different units",
			values:   []time.Duration{1 * time.Second, 500 * time.Millisecond, 1500 * time.Millisecond},
			expected: 1 * time.Second, // (1000 + 500 + 1500) / 3 = 1000ms
		},
		{
			name:     "microsecond precision",
			values:   []time.Duration{1 * time.Microsecond, 2 * time.Microsecond, 3 * time.Microsecond},
			expected: 2 * time.Microsecond,
		},
		{
			name:     "large values",
			values:   []time.Duration{10 * time.Minute, 20 * time.Minute},
			expected: 15 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateMean(tt.values)
			if result != tt.expected {
				t.Errorf("calculateMean() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCalculateMedian tests the CalculateMedian function with odd/even counts
func TestCalculateMedian(t *testing.T) {
	tests := []struct {
		name     string
		values   []time.Duration
		expected time.Duration
	}{
		{
			name:     "empty slice",
			values:   []time.Duration{},
			expected: 0,
		},
		{
			name:     "single value",
			values:   []time.Duration{100 * time.Millisecond},
			expected: 100 * time.Millisecond,
		},
		{
			name:     "two values",
			values:   []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			expected: 150 * time.Millisecond, // Interpolated median
		},
		{
			name:     "three values (odd)",
			values:   []time.Duration{100 * time.Millisecond, 300 * time.Millisecond, 200 * time.Millisecond},
			expected: 200 * time.Millisecond, // Middle value when sorted
		},
		{
			name:     "four values (even)",
			values:   []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond, 400 * time.Millisecond},
			expected: 250 * time.Millisecond, // Average of middle two values
		},
		{
			name:     "five values (odd)",
			values:   []time.Duration{500 * time.Millisecond, 100 * time.Millisecond, 300 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond},
			expected: 300 * time.Millisecond, // Middle value when sorted: [100, 200, 300, 400, 500]
		},
		{
			name:     "unsorted values",
			values:   []time.Duration{300 * time.Millisecond, 100 * time.Millisecond, 200 * time.Millisecond},
			expected: 200 * time.Millisecond,
		},
		{
			name:     "duplicate values",
			values:   []time.Duration{200 * time.Millisecond, 200 * time.Millisecond, 200 * time.Millisecond},
			expected: 200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateMedian(tt.values)
			if result != tt.expected {
				t.Errorf("calculateMedian() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCalculatePercentiles tests the CalculatePercentiles function with edge cases
func TestCalculatePercentiles(t *testing.T) {
	tests := []struct {
		name       string
		values     []time.Duration
		percentile float64
		expected   time.Duration
	}{
		{
			name:       "empty slice",
			values:     []time.Duration{},
			percentile: 50,
			expected:   0,
		},
		{
			name:       "single value P50",
			values:     []time.Duration{100 * time.Millisecond},
			percentile: 50,
			expected:   100 * time.Millisecond,
		},
		{
			name:       "single value P95",
			values:     []time.Duration{100 * time.Millisecond},
			percentile: 95,
			expected:   100 * time.Millisecond,
		},
		{
			name:       "two values P50",
			values:     []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			percentile: 50,
			expected:   150 * time.Millisecond,
		},
		{
			name:       "five values P95",
			values:     []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond, 400 * time.Millisecond, 500 * time.Millisecond},
			percentile: 95,
			expected:   480 * time.Millisecond, // 95th percentile with interpolation
		},
		{
			name:       "five values P99",
			values:     []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond, 400 * time.Millisecond, 500 * time.Millisecond},
			percentile: 99,
			expected:   496 * time.Millisecond, // 99th percentile with interpolation
		},
		{
			name:       "boundary P0",
			values:     []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond},
			percentile: 0,
			expected:   100 * time.Millisecond, // Minimum value
		},
		{
			name:       "boundary P100",
			values:     []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond},
			percentile: 100,
			expected:   300 * time.Millisecond, // Maximum value
		},
		{
			name:       "unsorted input",
			values:     []time.Duration{500 * time.Millisecond, 100 * time.Millisecond, 300 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond},
			percentile: 50,
			expected:   300 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a sorted copy since calculatePercentile expects sorted input
			sorted := make([]time.Duration, len(tt.values))
			copy(sorted, tt.values)
			if len(sorted) > 0 {
				// Sort the slice
				for i := 0; i < len(sorted)-1; i++ {
					for j := i + 1; j < len(sorted); j++ {
						if sorted[i] > sorted[j] {
							sorted[i], sorted[j] = sorted[j], sorted[i]
						}
					}
				}
			}

			result := calculatePercentile(sorted, tt.percentile)

			// Allow small tolerance for floating point arithmetic
			tolerance := 1 * time.Millisecond
			if abs(result-tt.expected) > tolerance {
				t.Errorf("calculatePercentile() = %v, want %v (tolerance: %v)", result, tt.expected, tolerance)
			}
		})
	}
}

// TestCalculateStdDev tests the CalculateStdDev function with uniform and varied data
func TestCalculateStdDev(t *testing.T) {
	tests := []struct {
		name       string
		values     []time.Duration
		mean       time.Duration
		expectZero bool // For cases where std dev should be 0
		tolerance  time.Duration
	}{
		{
			name:       "empty slice",
			values:     []time.Duration{},
			mean:       0,
			expectZero: true,
		},
		{
			name:       "single value",
			values:     []time.Duration{100 * time.Millisecond},
			mean:       100 * time.Millisecond,
			expectZero: true,
		},
		{
			name:       "two identical values",
			values:     []time.Duration{100 * time.Millisecond, 100 * time.Millisecond},
			mean:       100 * time.Millisecond,
			expectZero: true,
		},
		{
			name:       "uniform data (all same)",
			values:     []time.Duration{200 * time.Millisecond, 200 * time.Millisecond, 200 * time.Millisecond, 200 * time.Millisecond},
			mean:       200 * time.Millisecond,
			expectZero: true,
		},
		{
			name:      "simple case",
			values:    []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			mean:      150 * time.Millisecond,
			tolerance: 1 * time.Millisecond, // Allow for floating point precision
		},
		{
			name:      "varied data",
			values:    []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond, 400 * time.Millisecond},
			mean:      250 * time.Millisecond,
			tolerance: 5 * time.Millisecond,
		},
		{
			name:      "high variance",
			values:    []time.Duration{10 * time.Millisecond, 1000 * time.Millisecond},
			mean:      505 * time.Millisecond,
			tolerance: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateStandardDeviation(tt.values, tt.mean)

			if tt.expectZero {
				if result != 0 {
					t.Errorf("calculateStandardDeviation() = %v, want 0", result)
				}
			} else if tt.tolerance > 0 {
				// For non-zero cases, just verify result is reasonable (positive and within expected range)
				if result <= 0 {
					t.Errorf("calculateStandardDeviation() = %v, want positive value", result)
				}
				// Additional sanity check: std dev should not exceed the range of the data
				if len(tt.values) >= 2 {
					min, max := tt.values[0], tt.values[0]
					for _, v := range tt.values {
						if v < min {
							min = v
						}
						if v > max {
							max = v
						}
					}
					dataRange := max - min
					if result > dataRange {
						t.Errorf("calculateStandardDeviation() = %v exceeds data range %v", result, dataRange)
					}
				}
			}
		})
	}
}

// TestCalculateStats tests the main calculateStats function with comprehensive scenarios
func TestCalculateStats(t *testing.T) {
	tests := []struct {
		name   string
		values []time.Duration
	}{
		{
			name:   "empty slice",
			values: []time.Duration{},
		},
		{
			name:   "single value",
			values: []time.Duration{100 * time.Millisecond},
		},
		{
			name:   "two values",
			values: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
		},
		{
			name: "normal distribution",
			values: []time.Duration{
				90 * time.Millisecond, 95 * time.Millisecond, 100 * time.Millisecond,
				105 * time.Millisecond, 110 * time.Millisecond,
			},
		},
		{
			name: "skewed distribution",
			values: []time.Duration{
				10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond,
				40 * time.Millisecond, 500 * time.Millisecond, // Outlier
			},
		},
		{
			name: "microsecond precision",
			values: []time.Duration{
				1 * time.Microsecond, 2 * time.Microsecond, 3 * time.Microsecond,
				4 * time.Microsecond, 5 * time.Microsecond,
			},
		},
		{
			name: "large values",
			values: []time.Duration{
				1 * time.Minute, 2 * time.Minute, 3 * time.Minute,
				4 * time.Minute, 5 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := calculateStats(tt.values)
			mean, median, min, max, stddev, p95, p99 := s.Mean, s.Median, s.Min, s.Max, s.StdDev, s.P95, s.P99

			if len(tt.values) == 0 {
				// All values should be zero for empty slice
				if mean != 0 || median != 0 || min != 0 || max != 0 || stddev != 0 || p95 != 0 || p99 != 0 {
					t.Errorf("calculateStats() with empty slice should return all zeros, got mean=%v, median=%v, min=%v, max=%v, stddev=%v, p95=%v, p99=%v",
						s.Mean, s.Median, s.Min, s.Max, s.StdDev, s.P95, s.P99)
				}
				return
			}

			// Verify statistical constraints
			// Min should be <= all other values
			if min > mean || min > median || min > max || min > p95 || min > p99 {
				t.Errorf("Min (%v) should be <= all other stats", min)
			}

			// Max should be >= all other values
			if max < mean || max < median || max < min || max < p95 || max < p99 {
				t.Errorf("Max (%v) should be >= all other stats", max)
			}

			// P95 should be <= P99
			if p95 > p99 {
				t.Errorf("P95 (%v) should be <= P99 (%v)", p95, p99)
			}

			// Standard deviation should be non-negative
			if stddev < 0 {
				t.Errorf("Standard deviation (%v) should be non-negative", stddev)
			}

			// For single value, mean, median, min, max should all be equal
			if len(tt.values) == 1 {
				expected := tt.values[0]
				if mean != expected || median != expected || min != expected || max != expected {
					t.Errorf("For single value, mean, median, min, max should all equal %v, got mean=%v, median=%v, min=%v, max=%v",
						expected, mean, median, min, max)
				}
				if stddev != 0 {
					t.Errorf("Standard deviation should be 0 for single value, got %v", stddev)
				}
			}

			// Verify percentiles are within min-max range
			if p95 < min || p95 > max {
				t.Errorf("P95 (%v) should be within [%v, %v]", p95, min, max)
			}
			if p99 < min || p99 > max {
				t.Errorf("P99 (%v) should be within [%v, %v]", p99, min, max)
			}
		})
	}
}

// TestStatisticalAccuracy verifies statistical accuracy with known datasets
func TestStatisticalAccuracy(t *testing.T) {
	// Test with a known dataset where we can verify exact calculations
	values := []time.Duration{
		10 * time.Millisecond, // index 0
		20 * time.Millisecond, // index 1
		30 * time.Millisecond, // index 2
		40 * time.Millisecond, // index 3
		50 * time.Millisecond, // index 4
	}

	s := calculateStats(values)
	mean, median, min, max, stddev, p95, p99 := s.Mean, s.Median, s.Min, s.Max, s.StdDev, s.P95, s.P99

	// Verify exact values
	expectedMean := 30 * time.Millisecond // (10+20+30+40+50)/5 = 30
	if mean != expectedMean {
		t.Errorf("Mean = %v, want %v", mean, expectedMean)
	}

	expectedMedian := 30 * time.Millisecond // Middle value
	if median != expectedMedian {
		t.Errorf("Median = %v, want %v", median, expectedMedian)
	}

	expectedMin := 10 * time.Millisecond
	if min != expectedMin {
		t.Errorf("Min = %v, want %v", min, expectedMin)
	}

	expectedMax := 50 * time.Millisecond
	if max != expectedMax {
		t.Errorf("Max = %v, want %v", max, expectedMax)
	}

	// Standard deviation should be approximately 15.81 ms
	// σ = sqrt(((10-30)² + (20-30)² + (30-30)² + (40-30)² + (50-30)²) / (5-1))
	// σ = sqrt((400 + 100 + 0 + 100 + 400) / 4) = sqrt(1000/4) = sqrt(250) ≈ 15.81
	expectedStdDev := time.Duration(math.Sqrt(250) * float64(time.Millisecond))
	tolerance := 1 * time.Millisecond
	if abs(stddev-expectedStdDev) > tolerance {
		t.Errorf("StdDev = %v, want %v (±%v)", stddev, expectedStdDev, tolerance)
	}

	// P95 with 5 values: index = 0.95 * (5-1) = 3.8, so interpolate between index 3 and 4
	// P95 = 40 + 0.8 * (50-40) = 40 + 8 = 48
	expectedP95 := 48 * time.Millisecond
	if abs(p95-expectedP95) > tolerance {
		t.Errorf("P95 = %v, want %v (±%v)", p95, expectedP95, tolerance)
	}

	// P99 with 5 values: index = 0.99 * (5-1) = 3.96, so interpolate between index 3 and 4
	// P99 = 40 + 0.96 * (50-40) = 40 + 9.6 = 49.6
	expectedP99 := time.Duration(49.6 * float64(time.Millisecond))
	if abs(p99-expectedP99) > tolerance {
		t.Errorf("P99 = %v, want %v (±%v)", p99, expectedP99, tolerance)
	}
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("very large durations", func(t *testing.T) {
		values := []time.Duration{
			time.Duration(math.MaxInt64 / 4),
			time.Duration(math.MaxInt64 / 8),
			time.Duration(math.MaxInt64 / 16),
		}

		s := calculateStats(values)
		mean, median, min, max, stddev, p95, p99 := s.Mean, s.Median, s.Min, s.Max, s.StdDev, s.P95, s.P99

		// Use all values to avoid compilation errors
		_ = p95
		_ = p99

		// Should not panic and should return reasonable values
		if mean <= 0 || median <= 0 || min <= 0 || max <= 0 {
			t.Error("Stats should be positive for positive durations")
		}

		if stddev < 0 {
			t.Error("Standard deviation should be non-negative")
		}
	})

	t.Run("very small durations", func(t *testing.T) {
		values := []time.Duration{
			1 * time.Nanosecond,
			2 * time.Nanosecond,
			3 * time.Nanosecond,
		}

		s := calculateStats(values)
		mean, median, min, max, stddev, p95, p99 := s.Mean, s.Median, s.Min, s.Max, s.StdDev, s.P95, s.P99

		// Use all values to avoid compilation errors
		_ = stddev
		_ = p95
		_ = p99

		// Should handle nanosecond precision correctly
		if mean != 2*time.Nanosecond {
			t.Errorf("Mean = %v, want %v", mean, 2*time.Nanosecond)
		}
		if median != 2*time.Nanosecond {
			t.Errorf("Median = %v, want %v", median, 2*time.Nanosecond)
		}
		if min != 1*time.Nanosecond {
			t.Errorf("Min = %v, want %v", min, 1*time.Nanosecond)
		}
		if max != 3*time.Nanosecond {
			t.Errorf("Max = %v, want %v", max, 3*time.Nanosecond)
		}
	})

	t.Run("zero durations", func(t *testing.T) {
		values := []time.Duration{0, 0, 0}

		s := calculateStats(values)
		mean, median, min, max, stddev, p95, p99 := s.Mean, s.Median, s.Min, s.Max, s.StdDev, s.P95, s.P99

		if mean != 0 || median != 0 || min != 0 || max != 0 || stddev != 0 || p95 != 0 || p99 != 0 {
			t.Error("All stats should be zero for zero durations")
		}
	})

	t.Run("high variance data", func(t *testing.T) {
		data := []time.Duration{
			10 * time.Millisecond,
			50 * time.Millisecond,
			90 * time.Millisecond,
			30 * time.Millisecond,
			70 * time.Millisecond,
		}
		mean := calculateMean(data)
		p95 := calculatePercentile(data, 95)
		p99 := calculatePercentile(data, 99)

		// Verify that percentiles are reasonable
		if p95 <= mean {
			t.Errorf("Expected P95 %v to be greater than mean %v", p95, mean)
		}
		if p99 < p95 {
			t.Errorf("Expected P99 %v to be >= P95 %v", p99, p95)
		}
	})

	t.Run("uniform distribution", func(t *testing.T) {
		data := []time.Duration{
			50 * time.Millisecond,
			50 * time.Millisecond,
			50 * time.Millisecond,
			50 * time.Millisecond,
			50 * time.Millisecond,
		}
		mean := calculateMean(data)
		p95 := calculatePercentile(data, 95)
		p99 := calculatePercentile(data, 99)

		// All values should be the same
		expected := 50 * time.Millisecond
		if mean != expected {
			t.Errorf("Expected mean to be %v for uniform data, got %v", expected, mean)
		}
		if p95 != expected || p99 != expected {
			t.Errorf("Expected all percentiles to be %v, got P95=%v, P99=%v", expected, p95, p99)
		}
	})
}

// TestConcurrentStatsCalculation tests thread safety (if stats functions were to be called concurrently)
func TestConcurrentStatsCalculation(t *testing.T) {
	values := []time.Duration{
		100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond,
		400 * time.Millisecond, 500 * time.Millisecond,
	}

	const numGoroutines = 100
	results := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					results <- false
					return
				}
				results <- true
			}()

			// Calculate stats multiple times to check for race conditions
			for j := 0; j < 10; j++ {
				s := calculateStats(values)
				mean, median, min, max, stddev, p95, p99 := s.Mean, s.Median, s.Min, s.Max, s.StdDev, s.P95, s.P99

				// Use all values to avoid compilation errors
				_ = median

				// Verify basic constraints
				if min > max || p95 > p99 || stddev < 0 {
					panic("Invalid statistical result")
				}

				// Verify specific expected values for our test data
				expectedMean := 300 * time.Millisecond
				if mean != expectedMean {
					panic("Incorrect mean calculation")
				}
			}
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		success := <-results
		if !success {
			t.Error("Concurrent stats calculation failed")
		}
	}
}

// BenchmarkCalculateStats benchmarks the stats calculation performance
func BenchmarkCalculateStats(b *testing.B) {
	// Generate test data of different sizes
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		values := make([]time.Duration, size)
		for i := 0; i < size; i++ {
			values[i] = time.Duration(i) * time.Millisecond
		}

		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = calculateStats(values)
			}
		})
	}
}

// BenchmarkCalculateStandardDeviation benchmarks std dev calculation specifically
func BenchmarkCalculateStandardDeviation(b *testing.B) {
	values := make([]time.Duration, 1000)
	for i := 0; i < 1000; i++ {
		values[i] = time.Duration(i) * time.Millisecond
	}
	mean := 500 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateStandardDeviation(values, mean)
	}
}

// Helper function to calculate absolute difference between durations
func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
