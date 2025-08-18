// Package volume provides volume management for container runtimes.
// This file makes package installation instant and saves disk space
package volume

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mitl/internal/detector"
)

// testable exec wrapper
var execCommand = exec.Command

// Manager handles persistent volumes for caching dependencies
type Manager struct {
	runtime      string                    // docker, podman, etc. (path)
	projectRoot  string                    // Current project directory
	projectHash  string                    // Unique project identifier
	mu           sync.RWMutex              // Thread safety
	metadata     map[string]VolumeMetadata // Volume tracking
	metadataPath string                    // Path to metadata file
	pnpmStore    string                    // Global pnpm store volume name
}

// VolumeType represents different dependency types
type VolumeType string

const (
	VolumeTypeVendor      VolumeType = "vendor"       // PHP Composer
	VolumeTypePnpmStore   VolumeType = "pnpm-store"   // Global pnpm store
	VolumeTypePnpmModules VolumeType = "pnpm-modules" // Project node_modules
	VolumeTypePythonVenv  VolumeType = "venv"         // Python virtualenv
	VolumeTypeGoBuild     VolumeType = "go-build"     // Go build cache
	VolumeTypeRubyGems    VolumeType = "gems"         // Ruby gems
)

// VolumeMetadata tracks volume information
type VolumeMetadata struct {
	Name         string     `json:"name"`
	Type         VolumeType `json:"type"`
	ProjectPath  string     `json:"project_path"`
	LockfileHash string     `json:"lockfile_hash"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsed     time.Time  `json:"last_used"`
	Size         int64      `json:"size_bytes"`
	AccessCount  int        `json:"access_count"`
	Runtime      string     `json:"runtime"`
}

// NewManager creates a volume manager instance
func NewManager(runtime, projectRoot string) *Manager {
	if projectRoot == "" {
		cwd, _ := os.Getwd()
		projectRoot = cwd
	}
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.Getwd()
	}
	metaDir := filepath.Join(home, ".mitl")
	_ = os.MkdirAll(metaDir, 0o755)
	vm := &Manager{
		runtime:      runtime,
		projectRoot:  projectRoot,
		projectHash:  generateProjectHash(projectRoot),
		metadata:     make(map[string]VolumeMetadata),
		metadataPath: filepath.Join(metaDir, "volumes.json"),
		pnpmStore:    "mitl-pnpm-global-store",
	}
	vm.loadMetadata()
	_ = vm.ensurePnpmStore() // best effort
	return vm
}

// ensurePnpmStore creates the global pnpm store volume if it doesn't exist
func (vm *Manager) ensurePnpmStore() error {
	exists, err := vm.volumeExists(vm.pnpmStore)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Println("🏗️  Creating global pnpm store (one-time setup)...")
		cmd := execCommand(vm.runtime, "volume", "create", vm.pnpmStore)
		if err := cmd.Run(); err != nil {
			// Some runtimes may not support volumes; continue gracefully
			return fmt.Errorf("create pnpm store: %w", err)
		}
		vm.mu.Lock()
		vm.metadata[vm.pnpmStore] = VolumeMetadata{
			Name:        vm.pnpmStore,
			Type:        VolumeTypePnpmStore,
			ProjectPath: "",
			CreatedAt:   time.Now(),
			LastUsed:    time.Now(),
			Runtime:     vm.runtime,
		}
		vm.mu.Unlock()
		vm.saveMetadata()
		fmt.Println("✅ Global pnpm store created (shared across all projects)")
	}
	return nil
}

// GetMounts returns volume and env flags for container run
func (vm *Manager) GetMounts(projectType detector.ProjectType) []string {
	mounts := []string{}
	// Always mount source code
	mounts = append(mounts, "-v", fmt.Sprintf("%s:/app", vm.projectRoot))
	// Project-specific dependency volumes
	switch {
	case strings.HasPrefix(string(projectType), "php"):
		mounts = append(mounts, vm.getPHPMounts()...)
	case strings.HasPrefix(string(projectType), "node"):
		mounts = append(mounts, vm.getNodeMounts()...)
	case strings.HasPrefix(string(projectType), "python"):
		mounts = append(mounts, vm.getPythonMounts()...)
	case strings.HasPrefix(string(projectType), "go"):
		mounts = append(mounts, vm.getGoMounts()...)
	}
	return mounts
}

// getNodeMounts returns mounts for Node.js projects (enforcing pnpm)
func (vm *Manager) getNodeMounts() []string {
	mounts := []string{}
	// Global pnpm store
	mounts = append(mounts, "-v", fmt.Sprintf("%s:/root/.local/share/pnpm/store", vm.pnpmStore))
	// Project-specific node_modules
	modulesVolume := vm.getOrCreateVolume(VolumeTypePnpmModules)
	mounts = append(mounts, "-v", fmt.Sprintf("%s:/app/node_modules", modulesVolume))
	// Env to force pnpm store
	mounts = append(mounts, "-e", "PNPM_STORE_DIR=/root/.local/share/pnpm/store", "-e", "PNPM_PACKAGE_IMPORT_METHOD=hard-link")
	return mounts
}

// getPHPMounts returns mounts for PHP projects
func (vm *Manager) getPHPMounts() []string {
	vendorVolume := vm.getOrCreateVolume(VolumeTypeVendor)
	// Ensure global composer cache volume exists
	composerCache := "mitl-composer-cache"
	if ok, _ := vm.volumeExists(composerCache); !ok {
		_ = exec.Command(vm.runtime, "volume", "create", composerCache).Run()
	}
	return []string{
		"-v", fmt.Sprintf("%s:/app/vendor", vendorVolume),
		"-v", fmt.Sprintf("%s:/root/.composer/cache", composerCache),
	}
}

func (vm *Manager) getPythonMounts() []string {
	venvVolume := vm.getOrCreateVolume(VolumeTypePythonVenv)
	return []string{"-v", fmt.Sprintf("%s:/app/.venv", venvVolume)}
}

func (vm *Manager) getGoMounts() []string {
	// Go build cache can be shared; keep per-project for simplicity
	goVol := vm.getOrCreateVolume(VolumeTypeGoBuild)
	return []string{"-v", fmt.Sprintf("%s:/root/.cache/go-build", goVol)}
}

// getOrCreateVolume creates a volume if needed and returns its name
func (vm *Manager) getOrCreateVolume(volType VolumeType) string {
	lockfileHash := vm.calculateLockfileHash(volType)
	if lockfileHash == "" {
		lockfileHash = vm.projectHash[:12]
	}
	volumeName := fmt.Sprintf("mitl-%s-%s-%s", vm.projectHash[:8], volType, lockfileHash[:8])

	vm.mu.Lock()
	defer vm.mu.Unlock()

	if meta, ok := vm.metadata[volumeName]; ok {
		if meta.LockfileHash == lockfileHash {
			meta.LastUsed = time.Now()
			meta.AccessCount++
			vm.metadata[volumeName] = meta
			vm.saveMetadata()
			return volumeName
		}
		// Invalidate old volume if hash mismatch
		_ = vm.deleteVolume(volumeName)
		delete(vm.metadata, volumeName)
	}
	// Create
	if err := vm.createVolume(volumeName, volType, lockfileHash); err != nil {
		// If creation failed (runtime may not support volumes), return path mapping fallback
		return volumeName
	}
	return volumeName
}

// calculateLockfileHash generates hash from lock files
func (vm *Manager) calculateLockfileHash(volType VolumeType) string {
    var files []string
    switch volType {
    case VolumeTypeVendor:
        files = []string{"composer.lock"}
    case VolumeTypePnpmModules:
        files = []string{"pnpm-lock.yaml", "package.json"}
    case VolumeTypePnpmStore:
        // Global store not tied to project lockfiles; no hash input
        files = nil
    case VolumeTypePythonVenv:
        files = []string{"requirements.txt", "Pipfile.lock", "poetry.lock"}
    case VolumeTypeGoBuild:
        files = []string{"go.sum", "go.mod"}
    case VolumeTypeRubyGems:
        files = []string{"Gemfile.lock"}
    }
    if len(files) == 0 {
        return ""
    }
	h := sha256.New()
	for _, f := range files {
		p := filepath.Join(vm.projectRoot, f)
		if b, err := os.ReadFile(p); err == nil {
			_, _ = h.Write(b)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// InterceptNodeCommand converts npm/yarn commands to pnpm and ensures corepack activation
func (vm *Manager) InterceptNodeCommand(args []string) []string {
	if len(args) == 0 {
		return args
	}
	if strings.Contains(args[0], "pnpm") {
		return args
	}
	// helper to wrap into sh -lc with corepack
	wrap := func(cmd string) []string {
		msg := fmt.Sprintf("🔄 Enforcing pnpm: %s", cmd)
		fmt.Println(msg)
		// Ensure every 'pnpm ' invocation is prefixed with 'corepack '
		patched := strings.ReplaceAll(cmd, "pnpm ", "corepack pnpm ")
		full := "corepack prepare pnpm@latest --activate && " + patched
		return []string{"sh", "-lc", full}
	}
	switch args[0] {
	case "npm":
		if len(args) > 1 {
			sub := args[1]
			switch sub {
			case "ci":
				return wrap("pnpm install --frozen-lockfile || pnpm install --no-frozen-lockfile")
			case "install", "i":
				return wrap("pnpm install")
			case "run":
				return wrap("pnpm run " + strings.Join(args[2:], " "))
			case "test":
				return wrap("pnpm test")
			default:
				return wrap("pnpm " + strings.Join(args[1:], " "))
			}
		}
		return wrap("pnpm install")
	case "yarn":
		if len(args) > 1 {
			sub := args[1]
			switch sub {
			case "install":
				return wrap("pnpm install")
			case "add":
				return wrap("pnpm add " + strings.Join(args[2:], " "))
			case "remove":
				return wrap("pnpm remove " + strings.Join(args[2:], " "))
			default:
				return wrap("pnpm " + strings.Join(args[1:], " "))
			}
		}
		return wrap("pnpm install")
	default:
		return args
	}
}

// GetOrCreateVolume gets existing or creates new volume for dependencies
func (vm *Manager) GetOrCreateVolume(volType VolumeType, lockfileHash string) (string, bool, error) {
	volumeName := vm.generateVolumeName(volType, lockfileHash)

	exists, err := vm.volumeExists(volumeName)
	if err != nil {
		return "", false, err
	}

	if exists {
		vm.updateLastUsed(volumeName)
		return volumeName, true, nil
	}

	// Create new volume
	cmd := execCommand(vm.runtime, "volume", "create", volumeName)
	if err := cmd.Run(); err != nil {
		return "", false, fmt.Errorf("create volume: %w", err)
	}

	vm.mu.Lock()
	vm.metadata[volumeName] = VolumeMetadata{
		Name:         volumeName,
		Type:         volType,
		ProjectPath:  vm.projectRoot,
		LockfileHash: lockfileHash,
		CreatedAt:    time.Now(),
		LastUsed:     time.Now(),
		AccessCount:  1,
		Runtime:      vm.runtime,
	}
	vm.mu.Unlock()
	vm.saveMetadata()

	return volumeName, false, nil
}

// GetPnpmStoreMount returns mount flags for global pnpm store
func (vm *Manager) GetPnpmStoreMount() []string {
	return []string{
		"-v", fmt.Sprintf("%s:/root/.local/share/pnpm/store", vm.pnpmStore),
	}
}

// GetNodeModulesMount returns mount flags for project node_modules
func (vm *Manager) GetNodeModulesMount(lockfileHash string) ([]string, error) {
	volumeName, cached, err := vm.GetOrCreateVolume(VolumeTypePnpmModules, lockfileHash)
	if err != nil {
		return nil, err
	}

	if cached {
		fmt.Println("⚡ Using cached node_modules (instant install)")
	}

	return []string{
		"-v", fmt.Sprintf("%s:/app/node_modules", volumeName),
	}, nil
}

// CleanOldVolumes removes volumes not used for specified duration
func (vm *Manager) CleanOldVolumes(maxAge time.Duration) error {
	vm.mu.RLock()
	toDelete := []string{}
	cutoff := time.Now().Add(-maxAge)

	for name, meta := range vm.metadata {
		if meta.Type != VolumeTypePnpmStore && meta.LastUsed.Before(cutoff) {
			toDelete = append(toDelete, name)
		}
	}
	vm.mu.RUnlock()

	for _, name := range toDelete {
		fmt.Printf("🗑️  Removing old volume: %s\n", name)
		_ = execCommand(vm.runtime, "volume", "rm", name).Run()
		vm.mu.Lock()
		delete(vm.metadata, name)
		vm.mu.Unlock()
	}

	vm.saveMetadata()
	return nil
}

// Stats returns volume usage statistics
func (vm *Manager) Stats() map[string]interface{} {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_volumes": len(vm.metadata),
		"by_type":       make(map[VolumeType]int),
	}

	for _, meta := range vm.metadata {
		typeStats := stats["by_type"].(map[VolumeType]int)
		typeStats[meta.Type]++
	}

	return stats
}

// Metadata helpers
func (vm *Manager) loadMetadata() {
	b, err := os.ReadFile(vm.metadataPath)
	if err != nil {
		return
	}
	_ = json.Unmarshal(b, &vm.metadata)
}

func (vm *Manager) saveMetadata() {
	// Caller is responsible for synchronization; avoid taking locks here
	b, _ := json.MarshalIndent(vm.metadata, "", "  ")
	_ = os.WriteFile(vm.metadataPath, b, 0o644)
}

// Volume primitives
func (vm *Manager) volumeExists(name string) (bool, error) {
	cmd := execCommand(vm.runtime, "volume", "inspect", name)
	if err := cmd.Run(); err != nil {
		// Non-zero exit likely means not found or unsupported command
		// Try list grep fallback
		out, e := execCommand(vm.runtime, "volume", "ls").Output()
		if e != nil {
			return false, err
		}
		return strings.Contains(string(out), name), nil
	}
	return true, nil
}

func (vm *Manager) createVolume(name string, vt VolumeType, hash string) error {
	if err := execCommand(vm.runtime, "volume", "create", name).Run(); err != nil {
		return err
	}
	vm.metadata[name] = VolumeMetadata{
		Name:         name,
		Type:         vt,
		ProjectPath:  vm.projectRoot,
		LockfileHash: hash,
		CreatedAt:    time.Now(),
		LastUsed:     time.Now(),
		Runtime:      vm.runtime,
	}
	vm.saveMetadata()
	return nil
}

func (vm *Manager) deleteVolume(name string) error {
	_ = execCommand(vm.runtime, "volume", "rm", "-f", name).Run()
	delete(vm.metadata, name)
	vm.saveMetadata()
	return nil
}

// generateVolumeName creates a deterministic volume name
func (vm *Manager) generateVolumeName(volType VolumeType, lockfileHash string) string {
	return fmt.Sprintf("mitl-%s-%s-%s", volType, vm.projectHash[:8], lockfileHash[:8])
}

// updateLastUsed updates the last used timestamp
func (vm *Manager) updateLastUsed(name string) {
	vm.mu.Lock()
	if meta, ok := vm.metadata[name]; ok {
		meta.LastUsed = time.Now()
		meta.AccessCount++
		vm.metadata[name] = meta
	}
	vm.mu.Unlock()
}

// generateProjectHash creates a unique identifier for the project
func generateProjectHash(path string) string {
	h := sha256.Sum256([]byte(path))
	return hex.EncodeToString(h[:])
}
