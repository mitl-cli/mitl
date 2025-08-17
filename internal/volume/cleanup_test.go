package volume

import (
	"testing"
	"time"
)

func TestCleanup_ShowStatsAndClean(t *testing.T) {
	vm := NewManager("true", t.TempDir())
	// Seed metadata with a few volumes
	vm.metadata["v1"] = VolumeMetadata{Type: VolumeTypePnpmModules, LastUsed: time.Now().AddDate(0, 0, -40)}
	vm.metadata["v2"] = VolumeMetadata{Type: VolumeTypeVendor, LastUsed: time.Now()}
	vm.metadata["store"] = VolumeMetadata{Type: VolumeTypePnpmStore, LastUsed: time.Now().AddDate(0, 0, -400)}

	c := &VolumeCleanup{manager: vm, dryRun: true}
	c.ShowVolumeStats() // ensure no panic
	if err := c.CleanOldVolumes(30); err != nil {
		t.Fatalf("clean dry run: %v", err)
	}

	// Real delete run (should skip pnpm store)
	c.dryRun = false
	if err := c.CleanOldVolumes(30); err != nil {
		t.Fatalf("clean run: %v", err)
	}
}
