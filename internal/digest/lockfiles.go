// Package digest lockfile-specific hashing for dependency management.
// This module provides specialized hashing for different types of lockfiles to ensure
// that only meaningful dependency changes trigger cache invalidation, not cosmetic
// changes like timestamps or formatting differences.

package digest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// LockfileHasher handles specialized hashing for various lockfile formats.
// It extracts only the semantically meaningful parts of lockfiles to create
// stable hashes that don't change due to formatting or timestamp differences.
type LockfileHasher struct {
	root string // Project root directory
}

// NewLockfileHasher creates a new lockfile hasher for the specified directory.
func NewLockfileHasher(root string) *LockfileHasher {
	return &LockfileHasher{
		root: root,
	}
}

// HashLockfiles computes a combined hash of all lockfiles in the project.
// It looks for common lockfile types and applies specialized hashing to each.
func (lh *LockfileHasher) HashLockfiles() (string, error) {
	// Map of lockfile names to their specialized hash functions
	lockfiles := map[string]func([]byte) (string, error){
		"composer.lock":     lh.hashComposerLock,
		"package-lock.json": lh.hashPackageLock,
		"pnpm-lock.yaml":    lh.hashPnpmLock,
		"yarn.lock":         lh.hashYarnLock,
		"go.sum":            lh.hashGoSum,
		"go.mod":            lh.hashGoMod,
		"Gemfile.lock":      lh.hashGemfileLock,
		"requirements.txt":  lh.hashRequirements,
		"poetry.lock":       lh.hashPoetryLock,
		"Pipfile.lock":      lh.hashPipfileLock,
		"cargo.lock":        lh.hashCargoLock,
	}

	hasher := sha256.New()
	found := false

	// Process each lockfile type in alphabetical order for determinism
	var lockfileNames []string
	for name := range lockfiles {
		lockfileNames = append(lockfileNames, name)
	}
	sort.Strings(lockfileNames)

	for _, filename := range lockfileNames {
		hashFunc := lockfiles[filename]
		path := filepath.Join(lh.root, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			continue // File doesn't exist, skip
		}

		found = true

		// Use specialized hasher for this lockfile type
		hash, err := hashFunc(data)
		if err != nil {
			return "", fmt.Errorf("failed to hash %s: %w", filename, err)
		}

		// Add to combined hash with filename prefix for uniqueness
		fmt.Fprintf(hasher, "%s:%s\n", filename, hash)
	}

	if !found {
		return "no-lockfiles", nil
	}

	return hex.EncodeToString(hasher.Sum(nil))[:16], nil
}

// hashComposerLock extracts only dependency information from composer.lock.
// Ignores timestamps and metadata that don't affect actual dependencies.
func (lh *LockfileHasher) hashComposerLock(data []byte) (string, error) {
	var lock struct {
		Packages     []map[string]interface{} `json:"packages"`
		PackagesDev  []map[string]interface{} `json:"packages-dev"`
		ContentHash  string                   `json:"content-hash"`
		PlatformReqs map[string]string        `json:"platform"`
	}

	if err := json.Unmarshal(data, &lock); err != nil {
		// If can't parse JSON, fall back to raw hash
		return lh.hashRaw(data), nil
	}

	// Use Composer's own content-hash if available (most reliable)
	if lock.ContentHash != "" {
		return lock.ContentHash, nil
	}

	// Extract package dependencies manually
	hasher := sha256.New()

	// Hash production packages
	packages := lh.extractComposerPackages(lock.Packages)
	for _, pkg := range packages {
		fmt.Fprintln(hasher, pkg)
	}

	// Hash dev packages (separately to distinguish from production)
	devPackages := lh.extractComposerPackages(lock.PackagesDev)
	for _, pkg := range devPackages {
		fmt.Fprintf(hasher, "dev:%s\n", pkg)
	}

	// Hash platform requirements
	if lock.PlatformReqs != nil {
		var platReqs []string
		for name, version := range lock.PlatformReqs {
			platReqs = append(platReqs, fmt.Sprintf("%s:%s", name, version))
		}
		sort.Strings(platReqs)
		for _, req := range platReqs {
			fmt.Fprintf(hasher, "platform:%s\n", req)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// extractComposerPackages extracts package name and version pairs from Composer packages.
func (lh *LockfileHasher) extractComposerPackages(packages []map[string]interface{}) []string {
	var result []string
	for _, pkg := range packages {
		name, hasName := pkg["name"].(string)
		version, hasVersion := pkg["version"].(string)
		if hasName && hasVersion {
			result = append(result, fmt.Sprintf("%s@%s", name, version))
		}
	}
	sort.Strings(result)
	return result
}

// hashPackageLock handles npm package-lock.json files.
// Extracts dependency tree while ignoring resolved URLs that may change.
func (lh *LockfileHasher) hashPackageLock(data []byte) (string, error) {
	var lock struct {
		Name            string                            `json:"name"`
		Version         string                            `json:"version"`
		LockfileVersion int                               `json:"lockfileVersion"`
		Dependencies    map[string]map[string]interface{} `json:"dependencies"`
		Packages        map[string]map[string]interface{} `json:"packages"`
	}

	if err := json.Unmarshal(data, &lock); err != nil {
		return lh.hashRaw(data), nil
	}

	hasher := sha256.New()

	// Include lockfile version for compatibility
	fmt.Fprintf(hasher, "lockfileVersion:%d\n", lock.LockfileVersion)

	// Hash main package info
	if lock.Name != "" && lock.Version != "" {
		fmt.Fprintf(hasher, "main:%s@%s\n", lock.Name, lock.Version)
	}

	// For newer lockfile versions, use packages field
	if lock.Packages != nil {
		packages := lh.extractNpmPackages(lock.Packages)
		for _, pkg := range packages {
			fmt.Fprintln(hasher, pkg)
		}
	} else if lock.Dependencies != nil {
		// Fallback to dependencies field for older versions
		deps := lh.extractNpmDependencies(lock.Dependencies)
		for _, dep := range deps {
			fmt.Fprintln(hasher, dep)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// extractNpmPackages extracts package information from npm packages field.
func (lh *LockfileHasher) extractNpmPackages(packages map[string]map[string]interface{}) []string {
	var result []string
	for path, pkg := range packages {
		if path == "" { // Skip root package
			continue
		}

		version, hasVersion := pkg["version"].(string)
		if hasVersion {
			// Use path as package identifier (includes name and location)
			result = append(result, fmt.Sprintf("%s@%s", path, version))
		}
	}
	sort.Strings(result)
	return result
}

// extractNpmDependencies extracts dependencies from legacy npm format.
func (lh *LockfileHasher) extractNpmDependencies(deps map[string]map[string]interface{}) []string {
	var result []string

	var processDeps func(map[string]map[string]interface{}, string)
	processDeps = func(depMap map[string]map[string]interface{}, prefix string) {
		for name, dep := range depMap {
			version, hasVersion := dep["version"].(string)
			if hasVersion {
				fullName := prefix + name
				result = append(result, fmt.Sprintf("%s@%s", fullName, version))
			}

			// Process nested dependencies
			if nested, hasNested := dep["dependencies"].(map[string]map[string]interface{}); hasNested {
				processDeps(nested, prefix+name+"/")
			}
		}
	}

	processDeps(deps, "")
	sort.Strings(result)
	return result
}

// hashPnpmLock handles pnpm-lock.yaml files.
// Extracts package versions while ignoring volatile metadata.
func (lh *LockfileHasher) hashPnpmLock(data []byte) (string, error) {
	content := string(data)
	lines := strings.Split(content, "\n")
	hasher := sha256.New()

	// Extract lockfile version
	lockfileVersionRegex := regexp.MustCompile(`^lockfileVersion:\s*['"]?(\d+(?:\.\d+)?)['"]?`)

	// Extract package specifications (name and version)
	packageRegex := regexp.MustCompile(`^\s*['"]?([^:'"]+)['"]?:\s*(['"]?)([^'"]+)['"]?\s*$`)

	var packages []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Capture lockfile version
		if match := lockfileVersionRegex.FindStringSubmatch(line); len(match) > 1 {
			fmt.Fprintf(hasher, "lockfileVersion:%s\n", match[1])
			continue
		}

		// Capture package dependencies (skip comments and empty lines)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		if match := packageRegex.FindStringSubmatch(line); len(match) > 3 {
			name := match[1]
			version := match[3]

			// Filter out non-package lines (like registries, settings)
			if !strings.Contains(name, "registry") && !strings.Contains(name, "settings") &&
				!strings.Contains(name, "specifiers") {
				packages = append(packages, fmt.Sprintf("%s@%s", name, version))
			}
		}
	}

	// Sort packages for deterministic output
	sort.Strings(packages)
	for _, pkg := range packages {
		fmt.Fprintln(hasher, pkg)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// hashYarnLock handles yarn.lock files.
func (lh *LockfileHasher) hashYarnLock(data []byte) (string, error) {
	content := string(data)
	lines := strings.Split(content, "\n")
	hasher := sha256.New()

	// Extract yarn lockfile version
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# yarn lockfile v") {
		fmt.Fprintln(hasher, lines[0])
	}

	// Parse package entries
	packageRegex := regexp.MustCompile(`^"?([^"@]+)@([^"]+)"?:`)
	versionRegex := regexp.MustCompile(`^\s+version\s+"([^"]+)"`)

	var packages []string
	var currentPackage string

	for _, line := range lines {
		// Check for package declaration
		if match := packageRegex.FindStringSubmatch(line); len(match) > 2 {
			currentPackage = fmt.Sprintf("%s@%s", match[1], match[2])
		}

		// Check for version line
		if match := versionRegex.FindStringSubmatch(line); len(match) > 1 && currentPackage != "" {
			packages = append(packages, fmt.Sprintf("%s=%s", currentPackage, match[1]))
			currentPackage = ""
		}
	}

	sort.Strings(packages)
	for _, pkg := range packages {
		fmt.Fprintln(hasher, pkg)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// hashGoSum handles Go go.sum files (already deterministic).
func (lh *LockfileHasher) hashGoSum(data []byte) (string, error) {
	// go.sum is already deterministic and content-addressable
	return lh.hashRaw(data), nil
}

// hashGoMod handles Go go.mod files.
func (lh *LockfileHasher) hashGoMod(data []byte) (string, error) {
	// go.mod is deterministic, but we can extract just the require statements
	content := string(data)
	lines := strings.Split(content, "\n")
	hasher := sha256.New()

	requireRegex := regexp.MustCompile(`^\s*([^\s]+)\s+([^\s]+)`)
	inRequireBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}

		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		if strings.HasPrefix(line, "require ") || inRequireBlock {
			cleanLine := strings.TrimPrefix(line, "require ")
			if match := requireRegex.FindStringSubmatch(cleanLine); len(match) > 2 {
				fmt.Fprintf(hasher, "%s@%s\n", match[1], match[2])
			}
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// hashGemfileLock handles Ruby Gemfile.lock files.
func (lh *LockfileHasher) hashGemfileLock(data []byte) (string, error) {
	content := string(data)
	lines := strings.Split(content, "\n")
	hasher := sha256.New()

	gemRegex := regexp.MustCompile(`^\s+([a-zA-Z0-9_-]+)\s+\(([^)]+)\)`)
	inSpecsSection := false

	for _, line := range lines {
		if strings.TrimSpace(line) == "specs:" {
			inSpecsSection = true
			continue
		}

		if inSpecsSection && strings.HasPrefix(line, " ") {
			if match := gemRegex.FindStringSubmatch(line); len(match) > 2 {
				fmt.Fprintf(hasher, "%s@%s\n", match[1], match[2])
			}
		} else if inSpecsSection {
			// End of specs section
			break
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// hashRequirements handles Python requirements.txt files.
func (lh *LockfileHasher) hashRequirements(data []byte) (string, error) {
	content := string(data)
	lines := strings.Split(content, "\n")
	hasher := sha256.New()

	var requirements []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Skip pip install options
		if strings.HasPrefix(line, "-") {
			continue
		}

		requirements = append(requirements, line)
	}

	sort.Strings(requirements)
	for _, req := range requirements {
		fmt.Fprintln(hasher, req)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// hashPoetryLock handles Python poetry.lock files.
func (lh *LockfileHasher) hashPoetryLock(data []byte) (string, error) {
	// Poetry lock files are TOML format, but we can extract the essential info
	content := string(data)
	lines := strings.Split(content, "\n")
	hasher := sha256.New()

	packageRegex := regexp.MustCompile(`^name\s*=\s*"([^"]+)"`)
	versionRegex := regexp.MustCompile(`^version\s*=\s*"([^"]+)"`)

	var currentPackage string
	var packages []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if match := packageRegex.FindStringSubmatch(line); len(match) > 1 {
			currentPackage = match[1]
		}

		if match := versionRegex.FindStringSubmatch(line); len(match) > 1 && currentPackage != "" {
			packages = append(packages, fmt.Sprintf("%s@%s", currentPackage, match[1]))
			currentPackage = ""
		}
	}

	sort.Strings(packages)
	for _, pkg := range packages {
		fmt.Fprintln(hasher, pkg)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// hashPipfileLock handles Python Pipfile.lock files.
func (lh *LockfileHasher) hashPipfileLock(data []byte) (string, error) {
	var lock struct {
		Default map[string]map[string]interface{} `json:"default"`
		Develop map[string]map[string]interface{} `json:"develop"`
	}

	if err := json.Unmarshal(data, &lock); err != nil {
		return lh.hashRaw(data), nil
	}

	hasher := sha256.New()

	// Hash default dependencies
	if lock.Default != nil {
		deps := lh.extractPipfileDeps(lock.Default)
		for _, dep := range deps {
			fmt.Fprintln(hasher, dep)
		}
	}

	// Hash development dependencies
	if lock.Develop != nil {
		deps := lh.extractPipfileDeps(lock.Develop)
		for _, dep := range deps {
			fmt.Fprintf(hasher, "dev:%s\n", dep)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// extractPipfileDeps extracts dependencies from Pipfile.lock sections.
func (lh *LockfileHasher) extractPipfileDeps(deps map[string]map[string]interface{}) []string {
	var result []string
	for name, info := range deps {
		version, hasVersion := info["version"].(string)
		if hasVersion {
			result = append(result, fmt.Sprintf("%s@%s", name, version))
		}
	}
	sort.Strings(result)
	return result
}

// hashCargoLock handles Rust Cargo.lock files.
func (lh *LockfileHasher) hashCargoLock(data []byte) (string, error) {
	content := string(data)
	lines := strings.Split(content, "\n")
	hasher := sha256.New()

	nameRegex := regexp.MustCompile(`^name\s*=\s*"([^"]+)"`)
	versionRegex := regexp.MustCompile(`^version\s*=\s*"([^"]+)"`)

	var currentPackage string
	var packages []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if match := nameRegex.FindStringSubmatch(line); len(match) > 1 {
			currentPackage = match[1]
		}

		if match := versionRegex.FindStringSubmatch(line); len(match) > 1 && currentPackage != "" {
			packages = append(packages, fmt.Sprintf("%s@%s", currentPackage, match[1]))
			currentPackage = ""
		}
	}

	sort.Strings(packages)
	for _, pkg := range packages {
		fmt.Fprintln(hasher, pkg)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// hashRaw provides a fallback raw hash for unrecognized or unparseable files.
func (lh *LockfileHasher) hashRaw(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
