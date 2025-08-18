package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"mitl/pkg/terminal"
	"mitl/pkg/version"
)

// PanicHandler recovers from panics and shows friendly errors
type PanicHandler struct {
	errorHandler *ErrorHandler
}

// Setup installs the panic handler for the current scope
func (p *PanicHandler) Setup() { //nolint:revive,gocritic
    defer p.Recover() // Install panic recovery for the surrounding scope
}

// Recover catches panics and converts them to friendly output
func (p *PanicHandler) Recover() { //nolint:revive
	if r := recover(); r != nil {
		p.handlePanic(r)
	}
}

func (p *PanicHandler) handlePanic(r interface{}) {
	var message string
	switch v := r.(type) {
	case string:
		message = v
	case error:
		message = v.Error()
	default:
		message = fmt.Sprintf("%v", r)
	}

	stack := string(debug.Stack())
	crashReport := p.saveCrashReport(message, stack)

	fmt.Println()
	fmt.Printf("💥 %s%smitl crashed unexpectedly%s\n", terminal.Red, terminal.Bold, terminal.Reset)
	fmt.Println()
	fmt.Printf("Error: %s\n", message)
	fmt.Println()
	fmt.Printf("A crash report has been saved to:\n%s\n", crashReport)
	fmt.Println()
	fmt.Println("Please report this issue at:")
	fmt.Printf("%shttps://github.com/mitl-cli/mitl/issues%s\n", terminal.Cyan, terminal.Reset)
	fmt.Println()
	fmt.Println("Include the crash report and what you were doing when this happened.")

	os.Exit(2)
}

func (p *PanicHandler) saveCrashReport(message, stack string) string {
	crashDir := os.ExpandEnv("$HOME/.mitl/crashes")
	_ = os.MkdirAll(crashDir, 0o755)
	ts := time.Now().Format("2006-01-02-15-04-05")
	fp := filepath.Join(crashDir, fmt.Sprintf("crash-%s.txt", ts))
	report := fmt.Sprintf(`mitl Crash Report
==================
Time: %s
Version: %s
OS: %s
Arch: %s

Error:
%s

Stack Trace:
%s

Environment:
%s
`, time.Now().Format(time.RFC3339), version.Version, runtime.GOOS, runtime.GOARCH, message, stack, p.getEnvironmentInfo())
	_ = os.WriteFile(fp, []byte(report), 0o644)
	return fp
}

func (p *PanicHandler) getEnvironmentInfo() string {
	var info []string
	for _, key := range []string{"MITL_DEBUG", "MITL_RUNTIME", "DOCKER_HOST", "PATH"} {
		if v := os.Getenv(key); v != "" {
			info = append(info, fmt.Sprintf("%s=%s", key, v))
		}
	}
	return strings.Join(info, "\n")
}
