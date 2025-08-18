package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuntime_SelectByPerformanceAndNeedsBenchmark(t *testing.T) {
	rm := NewManager()
	// Override internals for test isolation
	rm.availableRuntimes = []Runtime{{Name: "r1", Path: "/bin/echo"}, {Name: "r2", Path: "/bin/echo"}}
	// Seed cache file
	dir := t.TempDir()
	rm.configPath = filepath.Join(dir, "bench.json")
	cf := benchmarkCacheFile{Hardware: rm.hardwareProfile, Results: map[string]BenchmarkResult{
		"r1": {Runtime: "r1", Score: 1.0, Timestamp: time.Now()},
		"r2": {Runtime: "r2", Score: 2.0, Timestamp: time.Now()},
	}}
	b, _ := json.Marshal(cf)
	os.WriteFile(rm.configPath, b, 0o644)

	rm.loadBenchmarkCache()
	best := rm.selectByPerformance()
	if best != "r1" {
		t.Fatalf("expected r1 best, got %s", best)
	}
	if rm.needsBenchmark() {
		t.Fatalf("did not expect benchmark needed with fresh cache")
	}
}

func TestRuntime_SaveCacheAndImageAvailable(t *testing.T) {
	rm := NewManager()
	rm.availableRuntimes = []Runtime{{Name: "echo", Path: "/bin/echo"}}
	dir := t.TempDir()
	rm.configPath = filepath.Join(dir, "bench.json")
	results := []BenchmarkResult{{Runtime: "echo", Score: 1.0, Timestamp: time.Now()}}
	rm.saveBenchmarkCache(results)
	// reload
	rm.loadBenchmarkCache()
	if _, ok := rm.benchmarkCache["echo"]; !ok {
		t.Fatalf("expected cached result")
	}
	// imageAvailable should return true with /bin/echo path
    rt := Runtime{Name: "echo", Path: "/bin/echo"}
    ok := rm.imageAvailable(&rt, "alpine:latest")
    if !ok {
        t.Fatalf("expected imageAvailable true with echo stub")
    }
}
