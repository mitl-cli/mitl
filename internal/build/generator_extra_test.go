package build

import (
	"testing"

	"mitl/internal/detector"
)

func TestGenerateDockerfile_OtherStacks(t *testing.T) {
	// Go
	d := detector.NewProjectDetector("")
	d.Type = detector.TypeGoModule
	g := NewDockerfileGenerator(d)
	if df, err := g.Generate(); err != nil || len(df) == 0 {
		t.Fatalf("go dockerfile failed: %v", err)
	}
	// Python
	d = detector.NewProjectDetector("")
	d.Type = detector.TypePythonGeneric
	d.Dependencies.Python.Version = "3.11"
	g = NewDockerfileGenerator(d)
	if df, err := g.Generate(); err != nil || len(df) == 0 {
		t.Fatalf("python dockerfile failed: %v", err)
	}
	// Generic
	if df, err := g.Generate(); err != nil || len(df) == 0 {
		t.Fatalf("generic dockerfile failed: %v", err)
	}
}

func TestOptimizationHints(t *testing.T) {
	d := detector.NewProjectDetector("")
	d.Type = detector.TypePHPLaravel
	g := NewDockerfileGenerator(d)
	hints := g.OptimizationHints()
	if len(hints) == 0 {
		t.Fatalf("expected hints for laravel")
	}
}
