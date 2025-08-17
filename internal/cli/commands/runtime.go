package commands

import (
	"fmt"

	"mitl/internal/container"
)

// Runtime handles runtime subcommands (info, benchmark, recommend).
// This command provides functionality to inspect and benchmark container runtimes.
func Runtime(args []string) error {
	if len(args) == 0 {
		args = []string{"info"}
	}
	rm := container.NewManager()
	switch args[0] {
	case "info":
		rm.ShowRuntimeInfo()
		return nil
	case "benchmark":
		// Parse flags: --include-build | --build | -b
		includeBuild := false
		if len(args) > 1 {
			for _, a := range args[1:] {
				switch a {
				case "--include-build", "--build", "-b":
					includeBuild = true
				}
			}
		}
		rm.ForceBenchmark(includeBuild)
		return nil
	case "recommend":
		rm.ShowRecommendations()
		return nil
	default:
		fmt.Println("Usage: mitl runtime [info|benchmark [--include-build]|recommend]")
		return fmt.Errorf("unknown runtime subcommand: %s", args[0])
	}
}
