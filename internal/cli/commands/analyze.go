package commands

import (
	"fmt"
	"os/exec"
	"strings"
)

// Analyze inspects the host for toolchains and prints a summary.
// This command detects PHP and Node.js versions and required extensions.
func Analyze(args []string) error {
	fmt.Println("=== Environment Analysis ===")
	detectPHP()
	detectNode()
	return nil
}

// runCommand executes a command and returns its stdout as a string.
func runCommand(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// detectPHP prints PHP version and required extensions.
func detectPHP() {
	phpVersion := runCommand("php", "-r", "echo PHP_VERSION;")
	fmt.Printf("PHP version: %s\n", phpVersion)
	modules := runCommand("php", "-m")
	required := []string{"intl", "mbstring", "pdo_sqlite", "pcntl"}
	for _, m := range required {
		if strings.Contains(modules, m) {
			fmt.Printf("Extension %s: found\n", m)
		} else {
			fmt.Printf("Extension %s: missing\n", m)
		}
	}
}

// detectNode prints Node and pnpm versions.
func detectNode() {
	nodeVersion := runCommand("node", "-v")
	fmt.Printf("Node version: %s\n", nodeVersion)
	pnpmVersion := runCommand("pnpm", "-v")
	if pnpmVersion == "" {
		pnpmVersion = runCommand("npm", "list", "-g", "pnpm")
	}
	if pnpmVersion != "" {
		fmt.Printf("pnpm version: %s\n", pnpmVersion)
	}
}
