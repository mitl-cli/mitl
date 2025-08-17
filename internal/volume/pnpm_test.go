package volume

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPnpmManager_ConvertAndStats(t *testing.T) {
	dir := t.TempDir()
	pm := &PnpmManager{projectRoot: dir}

	// No lock files
	if err := pm.ConvertToUsingPnpm(); err != nil {
		t.Fatalf("convert no-lock: %v", err)
	}
	s := pm.GetPnpmStats()
	if s.PercentSaved != 0 { // no lock yet
		t.Fatalf("expected 0 saved without lock, got %d", s.PercentSaved)
	}

	// NPM lock
	os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0644)
	if err := pm.ConvertToUsingPnpm(); err != nil {
		t.Fatalf("convert npm->pnpm: %v", err)
	}

	// Create pnpm lock and modules to affect stats
	os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: 9"), 0644)
	os.MkdirAll(filepath.Join(dir, "node_modules"), 0755)
	s = pm.GetPnpmStats()
	if s.PercentSaved == 0 {
		t.Fatalf("expected non-zero percent saved when pnpm lock present")
	}
}

func TestPnpm_InjectOptimizations(t *testing.T) {
	pm := &PnpmManager{projectRoot: t.TempDir()}
	lines := pm.InjectPnpmOptimizations()
	if len(lines) == 0 {
		t.Fatalf("expected optimization lines")
	}
}
