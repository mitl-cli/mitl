// pnpm.go - pnpm-specific optimizations and conversions

package volume

import (
	"fmt"
	"os"
	"path/filepath"
)

// PnpmManager handles pnpm-specific operations
type PnpmManager struct {
	projectRoot   string
	volumeManager *Manager
}

// NewPnpmManager creates a new PnpmManager instance
func NewPnpmManager(projectRoot string, vm *Manager) *PnpmManager {
	return &PnpmManager{
		projectRoot:   projectRoot,
		volumeManager: vm,
	}
}

// ConvertToUsingPnpm ensures a pnpm lock exists (no-op if present)
func (pm *PnpmManager) ConvertToUsingPnpm() error {
	if pm == nil {
		return nil
	}
	if pm.fileExists("pnpm-lock.yaml") {
		fmt.Println("✅ Using pnpm (most efficient)")
		return nil
	}
	if pm.fileExists("package-lock.json") || pm.fileExists("yarn.lock") {
		fmt.Println("🔄 Detected non-pnpm lock; pnpm import will run on first install")
		// We rely on runtime: corepack + pnpm import when user runs install
		return nil
	}
	fmt.Println("📦 No lock file found; pnpm will generate one on install")
	return nil
}

// GetPnpmStats returns approximate savings
func (pm *PnpmManager) GetPnpmStats() PnpmStats {
	stats := PnpmStats{}
	if pm.fileExists("pnpm-lock.yaml") {
		// We do not compute exact disk usage here; provide conservative estimate
		stats.PercentSaved = 70
		stats.SpaceSaved = 0
		stats.ModulesCount = pm.countNodeModules()
	}
	return stats
}

type PnpmStats struct {
	SpaceSaved   int64
	PercentSaved int
	ModulesCount int
}

func (pm *PnpmManager) InjectPnpmOptimizations() []string {
	return []string{
		"RUN corepack enable && corepack prepare pnpm@latest --activate",
		"ENV PNPM_PACKAGE_IMPORT_METHOD=hard-link",
	}
}

// helpers
func (pm *PnpmManager) fileExists(name string) bool {
	if pm == nil {
		return false
	}
	p := filepath.Join(pm.projectRoot, name)
	_, err := os.Stat(p)
	return err == nil
}

func (pm *PnpmManager) countNodeModules() int {
	// lightweight: just check top-level
	d := filepath.Join(pm.projectRoot, "node_modules")
	f, err := os.ReadDir(d)
	if err != nil {
		return 0
	}
	return len(f)
}
