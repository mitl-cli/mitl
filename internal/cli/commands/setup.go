package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Setup runs an interactive wizard that allows the user to choose
// a preferred container runtime. The chosen runtime is saved to the config
// file. If no runtimes are detected, an error message is printed and no
// configuration is saved.
func Setup(args []string) error {
	available := findAvailableCLIs()
	if len(available) == 0 {
		fmt.Println("No container runtimes detected. Install docker, podman, finch or container and re-run mitl setup.")
		return fmt.Errorf("no container runtimes found")
	}
	recommended := recommendCLI()
	fmt.Println("=== Mitl Setup Wizard ===")
	fmt.Println("Detected container runtimes:")
	for i, cli := range available {
		rec := ""
		if cli == recommended {
			rec = " (recommended)"
		}
		fmt.Printf("  %d) %s%s\n", i+1, cli, rec)
	}
	fmt.Println("\nSelect the number of the runtime you want to use for both building and running capsules.")
	fmt.Println("Press Enter to accept the recommended option.")
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Choice: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	idx := 0
	if input != "" {
		if v, err := strconv.Atoi(input); err == nil {
			idx = v
		}
		if idx < 1 || idx > len(available) {
			fmt.Println("Invalid choice; using recommended.")
			idx = 0
		}
	}
	var selected string
	if idx > 0 {
		selected = available[idx-1]
	} else {
		selected = recommended
	}
	cfg := Config{BuildCLI: selected, RunCLI: selected}
	saveConfig(cfg)
	fmt.Printf("Configured %s as the default container runtime. Configuration saved to %s\n", selected, configPath())
	return nil
}

// findAvailableCLIs returns a list of container runtime commands found in
// PATH, ordered by preference for the current OS. On macOS, the native
// container CLI (container) and finch are tried first. On other OSes,
// podman and nerdctl are preferred. Docker is always considered last.
func findAvailableCLIs() []string {
	var candidates []string
	if runtime.GOOS == "darwin" {
		candidates = []string{"container", "finch", "podman", "nerdctl", "docker"}
	} else {
		candidates = []string{"podman", "nerdctl", "docker"}
	}
	var available []string
	for _, cli := range candidates {
		if _, err := exec.LookPath(cli); err == nil {
			available = append(available, cli)
		}
	}
	return available
}

// recommendCLI chooses the best available runtime for the current host based
// on findAvailableCLIs. If no runtimes are found, docker is returned as
// a fallback.
func recommendCLI() string {
    avail := findAvailableCLIs()
    if len(avail) > 0 {
        return avail[0]
    }
    return "docker"
}
