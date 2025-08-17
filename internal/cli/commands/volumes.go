package commands

import (
	"fmt"

	"mitl/internal/volume"
)

// Volumes handles volume management commands (list, clean, stats, pnpm-stats).
// This command provides functionality to manage persistent dependency volumes.
func Volumes(args []string) error {
	if len(args) == 0 {
		args = []string{"list"}
	}
	vm := volume.NewManager(findRunCLI(), "")
	cleanup := volume.NewVolumeCleanup(vm)
	switch args[0] {
	case "list", "stats":
		cleanup.ShowVolumeStats()
		return nil
	case "clean":
		days := 30
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &days)
		}
		return cleanup.CleanOldVolumes(days)
	case "pnpm-stats":
		pnpm := volume.NewPnpmManager("", vm)
		s := pnpm.GetPnpmStats()
		fmt.Printf("🎉 pnpm estimated savings: %d%% (%d modules linked)\n", s.PercentSaved, s.ModulesCount)
		return nil
	default:
		fmt.Println("Usage: mitl volumes [list|clean [days]|stats|pnpm-stats]")
		return fmt.Errorf("unknown volumes subcommand: %s", args[0])
	}
}
