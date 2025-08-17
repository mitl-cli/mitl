package container

import "testing"

func TestRuntime_UpdateScoresAndSelectOptimal(t *testing.T) {
	rm := NewManager()
	rm.hardwareProfile.IsAppleSilicon = false
	rm.availableRuntimes = []Runtime{{Name: "r1", Path: "/bin/echo"}, {Name: "r2", Path: "/bin/echo"}}
	results := []BenchmarkResult{{Runtime: "r1", Score: 1.0}, {Runtime: "r2", Score: 2.0}}
	rm.updateRuntimeScores(results)
	sel := rm.SelectOptimal()
	if sel == "" {
		t.Fatalf("expected a selected path")
	}
}
