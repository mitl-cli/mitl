package container

import "testing"

func TestRuntime_BenchmarkAllExecOnly(t *testing.T) {
	rm := NewManager()
	rm.availableRuntimes = []Runtime{{Name: "echo", Path: "/bin/echo"}}
	rm.benchmarkAll(false)
	if _, ok := rm.benchmarkCache["echo"]; !ok {
		t.Fatalf("expected benchmark result for echo")
	}
}

func TestFileLockHelpers(t *testing.T) {
	lock := t.TempDir() + "/lock"
	if err := acquireFileLock(lock, 1000_000_000); err != nil { // 1s
		t.Fatalf("acquire lock: %v", err)
	}
	// second acquire should timeout quickly
	err := acquireFileLock(lock, 100_000_000) // 100ms
	if err == nil {
		t.Fatalf("expected timeout on second lock acquisition")
	}
	releaseFileLock(lock)
}

func TestRuntime_BenchmarkAllIncludeBuild(t *testing.T) {
	rm := NewManager()
	rm.availableRuntimes = []Runtime{{Name: "echo", Path: "/bin/echo"}}
	rm.benchmarkAll(true)
	if r, ok := rm.benchmarkCache["echo"]; !ok || r.Mode != "build+exec" {
		t.Fatalf("expected build+exec result, got %+v", r)
	}
}
