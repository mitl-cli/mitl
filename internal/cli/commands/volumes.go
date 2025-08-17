package commands

import (
	"fmt"
	"strconv"

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
			if v, err := strconv.Atoi(args[1]); err == nil && v >= 0 {
				days = v
			} else {
				fmt.Printf("Invalid days value '%s'; using default %d\n", args[1], days)
			}
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
