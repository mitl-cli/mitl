package container

import (
	"runtime"
	"testing"
)

func TestHardwareDetection(t *testing.T) {
	rm := NewManager()
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if !rm.hardwareProfile.IsAppleSilicon {
			t.Error("Failed to detect Apple Silicon")
		}
	}
}

func TestRuntimePriority(t *testing.T) {
	rm := NewManager()
	if rm.hardwareProfile.IsAppleSilicon {
		for _, rt := range rm.availableRuntimes {
			if rt.Name == "container" && rt.Priority != 100 {
				t.Errorf("Apple Container should have priority 100, got %d", rt.Priority)
			}
		}
	}
}

func TestBenchmarkNormalization(t *testing.T) {
	rm := NewManager()
	results := []BenchmarkResult{
		{Runtime: "fast", Score: 2.0},
		{Runtime: "slow", Score: 10.0},
	}
	rm.normalizeScores(results)
	var fast, slow float64
	for _, r := range results {
		if r.Runtime == "fast" {
			fast = r.Score
		}
		if r.Runtime == "slow" {
			slow = r.Score
		}
	}
	if fast != 1.0 {
		t.Fatalf("expected fastest score 1.0, got %v", fast)
	}
	if slow <= 1.0 {
		t.Fatalf("expected slower score > 1.0, got %v", slow)
	}
}

func BenchmarkRuntimeSelection(b *testing.B) {
	rm := NewManager()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rm.SelectOptimal()
	}
}
