package bench

import (
	"encoding/json"
	"time"
)

// Category defines benchmark categories
type Category string

const (
	// CategoryBuild represents build-time benchmarks
	CategoryBuild Category = "build"
	// CategoryRun represents runtime benchmarks
	CategoryRun Category = "run"
	// CategoryCache represents cache-related benchmarks
	CategoryCache Category = "cache"
	// CategoryVolume represents volume-related benchmarks
	CategoryVolume Category = "volume"
	// CategoryE2E represents end-to-end benchmarks
	CategoryE2E Category = "e2e"
)

// Duration wraps time.Duration with custom JSON marshaling
type Duration struct {
	time.Duration
}

// MarshalJSON implements json.Marshaler for Duration
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String())
}

// UnmarshalJSON implements json.Unmarshaler for Duration
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return nil
	}
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	AllocBytes      uint64 `json:"alloc_bytes"`
	TotalAllocBytes uint64 `json:"total_alloc_bytes"`
	SysBytes        uint64 `json:"sys_bytes"`
	NumGC           uint32 `json:"num_gc"`
	HeapAllocBytes  uint64 `json:"heap_alloc_bytes"`
	HeapSysBytes    uint64 `json:"heap_sys_bytes"`
	HeapIdleBytes   uint64 `json:"heap_idle_bytes"`
	HeapInuseBytes  uint64 `json:"heap_inuse_bytes"`
}

// Result represents the result of a benchmark run
type Result struct {
	Name        string      `json:"name"`
	Category    Category    `json:"category"`
	Description string      `json:"description"`
	Iterations  int         `json:"iterations"`
	Mean        Duration    `json:"mean"`
	Median      Duration    `json:"median"`
	Min         Duration    `json:"min"`
	Max         Duration    `json:"max"`
	StdDev      Duration    `json:"std_dev"`
	P95         Duration    `json:"p95"`
	P99         Duration    `json:"p99"`
	TotalTime   Duration    `json:"total_time"`
	Memory      MemoryStats `json:"memory,omitempty"`
	Success     bool        `json:"success"`
	Error       string      `json:"error,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
}

// Config represents benchmark execution configuration
type Config struct {
	MinIterations     int           `json:"min_iterations"`
	MaxIterations     int           `json:"max_iterations"`
	Duration          time.Duration `json:"duration"`
	WarmupIterations  int           `json:"warmup_iterations"`
	CooldownDuration  time.Duration `json:"cooldown_duration"`
	CollectMemoryInfo bool          `json:"collect_memory_info"`
	Parallel          bool          `json:"parallel"`
	Verbose           bool          `json:"verbose"`
}

// DefaultConfig returns a default benchmark configuration
func DefaultConfig() Config {
	return Config{
		MinIterations:     10,
		MaxIterations:     1000,
		Duration:          10 * time.Second,
		WarmupIterations:  3,
		CooldownDuration:  100 * time.Millisecond,
		CollectMemoryInfo: true,
		Parallel:          false,
		Verbose:           false,
	}
}
