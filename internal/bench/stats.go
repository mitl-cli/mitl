package bench

import (
	"math"
	"sort"
	"time"
)

// Stats aggregates statistical measurements for a series of durations.
type Stats struct {
	Mean   time.Duration
	Median time.Duration
	Min    time.Duration
	Max    time.Duration
	StdDev time.Duration
	P95    time.Duration
	P99    time.Duration
}

// calculateStats computes statistical measurements from a slice of durations
func calculateStats(durations []time.Duration) Stats {
	if len(durations) == 0 {
		return Stats{}
	}

	// Make a copy and sort for percentile calculations
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Min and Max
	minDur := sorted[0]
	maxDur := sorted[len(sorted)-1]

	// Mean
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	mean := sum / time.Duration(len(durations))

	// Median
	median := calculatePercentile(sorted, 50)

	// Percentiles
	p95 := calculatePercentile(sorted, 95)
	p99 := calculatePercentile(sorted, 99)

	// Standard deviation
	stddev := calculateStandardDeviation(durations, mean)

	return Stats{
		Mean:   mean,
		Median: median,
		Min:    minDur,
		Max:    maxDur,
		StdDev: stddev,
		P95:    p95,
		P99:    p99,
	}
}

// calculatePercentile computes the nth percentile from a sorted slice of durations
func calculatePercentile(sorted []time.Duration, percentile float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	if len(sorted) == 1 {
		return sorted[0]
	}

	// Calculate the index for the percentile
	index := (percentile / 100.0) * float64(len(sorted)-1)

	// If index is a whole number, return that element
	if index == math.Floor(index) {
		return sorted[int(index)]
	}

	// Otherwise, interpolate between the two nearest values
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if upper >= len(sorted) {
		upper = len(sorted) - 1
	}

	lowerValue := float64(sorted[lower])
	upperValue := float64(sorted[upper])
	weight := index - math.Floor(index)

	interpolated := lowerValue + weight*(upperValue-lowerValue)
	return time.Duration(interpolated)
}

// calculateStandardDeviation computes the standard deviation of durations
func calculateStandardDeviation(durations []time.Duration, mean time.Duration) time.Duration {
	if len(durations) <= 1 {
		return 0
	}

	var sumSquaredDiffs float64
	meanFloat := float64(mean)

	for _, d := range durations {
		diff := float64(d) - meanFloat
		sumSquaredDiffs += diff * diff
	}

	variance := sumSquaredDiffs / float64(len(durations)-1)
	return time.Duration(math.Sqrt(variance))
}

// calculateMean computes the arithmetic mean of durations
func calculateMean(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return sum / time.Duration(len(durations))
}

// calculateMedian computes the median of durations
func calculateMedian(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	return calculatePercentile(sorted, 50)
}
