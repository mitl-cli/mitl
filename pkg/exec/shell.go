package exec

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Shell runs a shell command
func Shell(command string) error {
	shell := getShell()
	return exec.Command(shell, "-c", command).Run()
}

// ShellOutput runs a shell command and returns output
func ShellOutput(command string) (string, error) {
	shell := getShell()
	output, err := exec.Command(shell, "-c", command).Output()
	return strings.TrimSpace(string(output)), err
}

// ShellInteractive runs an interactive shell command
func ShellInteractive(command string) error {
	shell := getShell()
	cmd := exec.Command(shell, "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getShell returns the appropriate shell for the platform
func getShell() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}

	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	return "/bin/sh"
}

// Quote quotes a string for shell execution
func Quote(s string) string {
    if runtime.GOOS == "windows" {
        return fmt.Sprintf("%q", strings.ReplaceAll(s, `"`, `""`))
    }
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "'\\''"))
}

// JoinArgs joins arguments for shell execution
func JoinArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = Quote(arg)
	}
	return strings.Join(quoted, " ")
}
