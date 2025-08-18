// Package doctor provides system health checks for mitl.
package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// execCommand enables test stubbing.
var execCommand = exec.Command

// Doctor performs comprehensive system health checks
type Doctor struct {
	checks  []HealthCheck
	verbose bool
}

// HealthCheck represents a single diagnostic check
type HealthCheck interface {
	Name() string
	Description() string
	Run() CheckResult
	CanAutoFix() bool
	Fix() error
	Severity() Severity
}

// CheckResult contains the outcome of a health check
type CheckResult struct {
	Status     Status
	Message    string
	Details    string
	FixCommand string
	Impact     string
}

// Status represents check status
type Status int

const (
	StatusOK Status = iota
	StatusWarning
	StatusError
	StatusCritical
)

// Severity indicates how important a fix is
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// HealthReport summarizes checks
type HealthReport struct {
	TotalChecks int
	Passed      int
	Warnings    int
	Errors      int
	Critical    int
	StartTime   time.Time
	EndTime     time.Time
}

// Run executes all checks and prints a concise report
func (d *Doctor) Run() HealthReport {
	d.checks = []HealthCheck{
		&RuntimeCheck{},
		&DiskSpaceCheck{},
		&PermissionsCheck{},
		&CacheHealthCheck{},
		&PnpmOptimizationCheck{},
	}
	rpt := HealthReport{StartTime: time.Now()}
	fmt.Println("\n🏹 mitl doctor - System Health Check")
	fmt.Println(strings.Repeat("=", 52))
	for _, c := range d.checks {
		res := c.Run()
		d.printResult(res)
		rpt.TotalChecks++
		switch res.Status {
		case StatusOK:
			rpt.Passed++
		case StatusWarning:
			rpt.Warnings++
		case StatusError:
			rpt.Errors++
		case StatusCritical:
			rpt.Critical++
		}
	}
	rpt.EndTime = time.Now()
	// Simple performance score: 100 minus penalties
	score := 100
	score -= rpt.Warnings * 5
	score -= rpt.Errors * 15
	score -= rpt.Critical * 25
	if score < 0 {
		score = 0
	}
	fmt.Printf("\n⏱  Completed in %.2fs\n", rpt.EndTime.Sub(rpt.StartTime).Seconds())
	fmt.Printf("Performance Score: %d/100\n", score)
	fmt.Println("Run 'mitl doctor --fix' to auto-fix issues where possible")
	return rpt
}

func (d *Doctor) printResult(r CheckResult) {
    icon := "✅"
    switch r.Status {
    case StatusOK:
        // keep default icon
    case StatusWarning:
        icon = "⚠️ "
    case StatusError, StatusCritical:
        icon = "❌"
    }
	fmt.Printf("%s %s\n", icon, r.Message)
	if r.Details != "" && d.verbose {
		fmt.Printf("   %s\n", r.Details)
	}
	if r.FixCommand != "" && r.Status != StatusOK {
		fmt.Printf("   💡 Fix: %s\n", r.FixCommand)
	}
	if r.Impact != "" && r.Status == StatusCritical {
		fmt.Printf("   ⚠️  Impact: %s\n", r.Impact)
	}
}

// RuntimeCheck verifies container runtime availability and performance
type RuntimeCheck struct{}

func (r *RuntimeCheck) Name() string        { return "Container Runtime" }
func (r *RuntimeCheck) Description() string { return "Checking for optimal container runtime" }

func (r *RuntimeCheck) Run() CheckResult {
	result := CheckResult{}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if _, err := exec.LookPath("container"); err != nil {
			result.Status = StatusWarning
			result.Message = "Apple Container not found (5-10x faster than Docker)"
			result.Details = "Docker Desktop is slower on Apple Silicon"
			result.FixCommand = "Download from: developer.apple.com/virtualization"
			result.Impact = "Builds and runs are 5-10x slower than optimal"
			return result
		}
	}
	// Any runtime available?
	for _, rt := range []string{"container", "finch", "podman", "docker"} {
		if _, err := exec.LookPath(rt); err == nil {
			// Is it responding?
			if err := execCommand(rt, "version").Run(); err != nil {
				return CheckResult{Status: StatusError, Message: fmt.Sprintf("%s is not responding", rt), Details: "Runtime installed but not running", FixCommand: fmt.Sprintf("%s start", rt), Impact: "Commands will fail"}
			}
			return CheckResult{Status: StatusOK, Message: fmt.Sprintf("Using %s (healthy)", rt)}
		}
	}
	return CheckResult{Status: StatusCritical, Message: "No container runtime found", Details: "mitl requires Docker/Podman/etc.", FixCommand: "brew install docker || brew install podman", Impact: "mitl cannot run"}
}
func (r *RuntimeCheck) CanAutoFix() bool   { return false }
func (r *RuntimeCheck) Fix() error         { return nil }
func (r *RuntimeCheck) Severity() Severity { return SeverityCritical }

// DiskSpaceCheck ensures sufficient disk space
type DiskSpaceCheck struct{}

func (d *DiskSpaceCheck) Name() string        { return "Disk Space" }
func (d *DiskSpaceCheck) Description() string { return "Checking available disk" }
func (d *DiskSpaceCheck) CanAutoFix() bool    { return false }
func (d *DiskSpaceCheck) Fix() error          { return nil }
func (d *DiskSpaceCheck) Severity() Severity  { return SeverityMedium }

func (d *DiskSpaceCheck) Run() CheckResult {
	cmd := execCommand("df", "-h", "/")
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{Status: StatusWarning, Message: "Could not check disk space"}
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > 1 {
		f := strings.Fields(lines[1])
		if len(f) > 3 {
			var size float64
			var unit string
			if n, err := fmt.Sscanf(f[3], "%f%s", &size, &unit); err == nil && n == 2 {
				if unit == "G" && size < 5 {
					return CheckResult{Status: StatusWarning, Message: fmt.Sprintf("Low disk space: %.1fGB free", size), FixCommand: "mitl volumes clean 30", Impact: "Builds may fail"}
				}
			}
		}
	}
	return CheckResult{Status: StatusOK, Message: "Sufficient disk space available"}
}

// PermissionsCheck verifies file and socket permissions
type PermissionsCheck struct{}

func (p *PermissionsCheck) Name() string        { return "Permissions" }
func (p *PermissionsCheck) Description() string { return "Checking permissions" }
func (p *PermissionsCheck) CanAutoFix() bool    { return true }
func (p *PermissionsCheck) Fix() error {
	// Ensure ~/.mitl exists with 0700 permissions
	cfgDir := filepath.Join(os.Getenv("HOME"), ".mitl")
	if _, err := os.Stat(cfgDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cfgDir, 0o700); err != nil {
			return err
		}
		return nil
	}
	// Correct permissions if needed
	if err := os.Chmod(cfgDir, 0o700); err != nil {
		return err
	}
	return nil
}
func (p *PermissionsCheck) Severity() Severity { return SeverityMedium }

func (p *PermissionsCheck) Run() CheckResult {
	socketPath := "/var/run/docker.sock"
	if _, err := os.Stat(socketPath); err == nil {
		if err := execCommand("docker", "ps").Run(); err != nil {
			return CheckResult{Status: StatusError, Message: "Cannot access Docker socket", Details: "Permission denied to /var/run/docker.sock", FixCommand: "sudo usermod -aG docker $USER && newgrp docker", Impact: "mitl commands require sudo"}
		}
	}
	// Config dir permissions (~/.mitl)
	cfgDir := filepath.Join(os.Getenv("HOME"), ".mitl")
	if info, err := os.Stat(cfgDir); err == nil {
		if info.Mode().Perm()&0o700 == 0 {
			return CheckResult{Status: StatusWarning, Message: "Config directory has incorrect permissions", FixCommand: "chmod 700 ~/.mitl", Impact: "Config may not save"}
		}
	}
	return CheckResult{Status: StatusOK, Message: "All permissions correct"}
}

// CacheHealthCheck verifies cache integrity
type CacheHealthCheck struct{}

func (c *CacheHealthCheck) Name() string        { return "Cache" }
func (c *CacheHealthCheck) Description() string { return "Checking capsule cache" }
func (c *CacheHealthCheck) CanAutoFix() bool    { return true }
func (c *CacheHealthCheck) Fix() error {
	// Attempt to clean old capsules via mitl CLI if available
	if _, err := exec.LookPath("mitl"); err == nil {
		// Clean old cache (default policy)
		return execCommand("mitl", "cache", "clean").Run()
	}
	return nil
}
func (c *CacheHealthCheck) Severity() Severity { return SeverityLow }

func (c *CacheHealthCheck) Run() CheckResult {
	cli := selectRuntime()
	out, _ := execCommand(cli, "images", "--filter", "reference=mitl-capsule:*", "--format", "table").Output()
	lines := strings.Split(string(out), "\n")
	count := len(lines) - 2
	if count > 50 {
		return CheckResult{Status: StatusWarning, Message: fmt.Sprintf("High cache usage: %d capsules", count), FixCommand: "mitl cache clean", Impact: "Excessive disk usage"}
	}
	if count < 0 {
		count = 0
	}
	return CheckResult{Status: StatusOK, Message: fmt.Sprintf("Cache healthy: %d capsules", count)}
}

// PnpmOptimizationCheck ensures pnpm is being used
type PnpmOptimizationCheck struct{}

func (p *PnpmOptimizationCheck) Name() string        { return "Node.js" }
func (p *PnpmOptimizationCheck) Description() string { return "Checking pnpm optimization" }
func (p *PnpmOptimizationCheck) CanAutoFix() bool    { return false }
func (p *PnpmOptimizationCheck) Fix() error          { return nil }
func (p *PnpmOptimizationCheck) Severity() Severity  { return SeverityMedium }

func (p *PnpmOptimizationCheck) Run() CheckResult {
	if _, err := os.Stat("package.json"); err != nil {
		return CheckResult{Status: StatusOK, Message: "Not a Node.js project"}
	}
	if _, err := exec.LookPath("pnpm"); err != nil {
		return CheckResult{Status: StatusWarning, Message: "pnpm not installed (70% space savings)", FixCommand: "npm install -g pnpm", Impact: "Using 3x more disk space"}
	}
	if _, err := os.Stat("pnpm-lock.yaml"); err != nil {
		return CheckResult{Status: StatusWarning, Message: "Project not using pnpm", FixCommand: "pnpm import && rm package-lock.json", Impact: "Missing performance gains"}
	}
	return CheckResult{Status: StatusOK, Message: "Using pnpm (optimal)"}
}

// Fix attempts automatic fixes for checks that support it.
func (d *Doctor) Fix() {
	fmt.Println("\n🔧 Attempting to fix issues...")
	for _, c := range d.checks {
		res := c.Run()
		if res.Status != StatusOK && c.CanAutoFix() {
			if err := c.Fix(); err != nil {
				fmt.Printf("❌ %s: fix failed: %v\n", c.Name(), err)
			} else {
				fmt.Printf("✅ %s: fixed\n", c.Name())
			}
		}
	}
}

// RunDoctorWithOptions runs checks and optionally applies fixes.
func RunDoctorWithOptions(verbose, fix bool) {
	d := &Doctor{verbose: verbose}
	_ = d.Run()
	if fix {
		d.Fix()
	}
}

// selectRuntime picks a runtime fallback when building simple doctor checks.
func selectRuntime() string {
	for _, rt := range []string{"container", "finch", "podman", "docker"} {
		if _, err := exec.LookPath(rt); err == nil {
			return rt
		}
	}
	return "docker"
}
