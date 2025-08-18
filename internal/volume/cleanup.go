// cleanup.go - Volume cleanup and lifecycle management

package volume

import (
	"fmt"
	"time"
)

// VolumeCleanup handles cleanup of old/unused volumes
type VolumeCleanup struct {
	manager *Manager
	dryRun  bool
	verbose bool
}

// NewVolumeCleanup creates a new VolumeCleanup instance
func NewVolumeCleanup(manager *Manager) *VolumeCleanup {
	return &VolumeCleanup{
		manager: manager,
		dryRun:  false,
		verbose: false,
	}
}

// CleanOldVolumes removes volumes not used in X days
func (vc *VolumeCleanup) CleanOldVolumes(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	var toDelete []string
	var spaceSaved int64

	fmt.Printf("🧹 Scanning for volumes unused for %d+ days...\n", days)

    for name := range vc.manager.metadata {
        meta := vc.manager.metadata[name]
        if meta.Type == VolumeTypePnpmStore { // never delete global store here
            continue
        }
        if meta.LastUsed.Before(cutoff) {
            toDelete = append(toDelete, name)
            spaceSaved += meta.Size
        }
    }

	if len(toDelete) == 0 {
		fmt.Println("✨ No old volumes to clean")
		return nil
	}

	fmt.Printf("Found %d volumes to clean (%.2f GB)\n", len(toDelete), float64(spaceSaved)/1024/1024/1024)

	if vc.dryRun {
		fmt.Println("Dry run - no volumes deleted")
		for _, v := range toDelete {
			fmt.Printf("  Would delete: %s\n", v)
		}
		return nil
	}

	for _, v := range toDelete {
		if err := vc.manager.deleteVolume(v); err != nil {
			fmt.Printf("⚠️  Failed to delete %s: %v\n", v, err)
		} else {
			fmt.Printf("  Deleted: %s\n", v)
		}
	}

	fmt.Printf("✅ Freed %.2f GB of disk space\n", float64(spaceSaved)/1024/1024/1024)
	return nil
}

// ShowVolumeStats displays volume usage statistics
func (vc *VolumeCleanup) ShowVolumeStats() {
	stats := struct {
		Total     int
		TotalSize int64
		ByType    map[VolumeType]int
	}{ByType: make(map[VolumeType]int)}

	for _, m := range vc.manager.metadata {
		stats.Total++
		stats.TotalSize += m.Size
		stats.ByType[m.Type]++
	}
	fmt.Println("=== Mitl Volume Statistics ===")
	fmt.Printf("Total volumes: %d\n", stats.Total)
	fmt.Printf("Total size: %.2f GB\n", float64(stats.TotalSize)/1024/1024/1024)
	fmt.Println("\nBy type:")
	for t, c := range stats.ByType {
		fmt.Printf("  %s: %d volumes\n", t, c)
	}
	if stats.ByType[VolumeTypePnpmModules] > 0 {
		saved := stats.ByType[VolumeTypePnpmModules] * 200 * 1024 * 1024 // rough est.
		fmt.Printf("\n🎉 pnpm is saving ~%.1f GB vs npm\n", float64(saved)/1024/1024/1024)
	}
}
