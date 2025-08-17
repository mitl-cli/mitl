package commands

import (
	"fmt"
	"os"
	"strings"
)

// Cache handles cache management commands (list, clean, stats).
// This command provides functionality to manage cached container images.
func Cache(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: mitl cache [list|clean|stats]")
		return fmt.Errorf("no cache subcommand specified")
	}
	switch args[0] {
	case "list":
		return listCachedCapsules()
	case "clean":
		return cleanOldCapsules()
	case "stats":
		return showCacheStatistics()
	default:
		fmt.Println("Usage: mitl cache [list|clean|stats]")
		return fmt.Errorf("unknown cache subcommand: %s", args[0])
	}
}

// listCachedCapsules lists all cached mitl capsule images.
func listCachedCapsules() error {
	runtime := findBuildCLI()
	cmd := execCommand(runtime, "images")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}
	fmt.Println("Cached capsules:")
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "mitl-capsule") {
			fmt.Println(line)
		}
	}
	return nil
}

// cleanOldCapsules removes all cached mitl capsule images.
func cleanOldCapsules() error {
	runtime := findBuildCLI()
	cmd := execCommand(runtime, "images", "-q", "mitl-capsule")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to query images: %w", err)
	}
	ids := strings.Fields(string(out))
	if len(ids) == 0 {
		fmt.Println("No cached capsules found.")
		return nil
	}
	args := append([]string{"rmi"}, ids...)
	rm := execCommand(runtime, args...)
	rm.Stdout = os.Stdout
	rm.Stderr = os.Stderr
	if err := rm.Run(); err != nil {
		return fmt.Errorf("failed to remove images: %w", err)
	}
	return nil
}

// showCacheStatistics shows statistics about cached capsule images.
func showCacheStatistics() error {
	runtime := findBuildCLI()
	cmd := execCommand(runtime, "images", "-q", "mitl-capsule")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to query images: %w", err)
	}
	ids := strings.Fields(string(out))
	fmt.Printf("Cached capsules: %d\n", len(ids))
	return nil
}
