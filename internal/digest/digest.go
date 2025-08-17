// Package digest provides cryptographic hashing utilities for project state management.
// It generates deterministic digests with .mitlignore support to enable intelligent caching
// and determine when containers need rebuilding.
//
// The enhanced digest system ensures 100% deterministic hashing across platforms by:
//   - Content normalization (line endings, BOM removal)
//   - .mitlignore pattern support for file exclusion
//   - Cross-platform path handling
//   - Specialized lockfile hashing
//
// This package is critical for mitl's performance optimization, ensuring that containers
// are only rebuilt when dependencies actually change, not on every invocation.
package digest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Digest represents a complete project digest with metadata about the calculation.
// This struct provides complete information about what files were included
// and how the digest was calculated for transparency and debugging.
type Digest struct {
	Hash      string       `json:"hash"`       // The calculated digest value
	Algorithm string       `json:"algorithm"`  // Hash algorithm used (sha256, blake3)
	Timestamp time.Time    `json:"timestamp"`  // When the digest was calculated
	FileCount int          `json:"file_count"` // Number of files included
	TotalSize int64        `json:"total_size"` // Total size of all files
	Files     []FileDigest `json:"files"`      // Files included in the digest
	Options   Options      `json:"options"`    // Configuration used for calculation
}

// FileDigest contains metadata for a single file included in the digest calculation.
// This provides transparency about what files affect the cache and their characteristics.
type FileDigest struct {
	Path string `json:"path"` // Relative path from project root
	Hash string `json:"hash"` // Individual file hash
	Size int64  `json:"size"` // File size in bytes
}

// ProjectCalculator computes deterministic digests for projects with full configurability.
// It integrates the existing calculator infrastructure with the new interface.
type ProjectCalculator struct {
	root          string
	ignoreMatcher *IgnoreRules
	options       Options
	internalCalc  *Calculator // Reference to existing calculator.Calculator
}

// Options configures digest calculation behavior to meet different use cases.
type Options struct {
	Algorithm      string   `json:"algorithm"`       // Hash algorithm: "sha256", "blake3" (default: "blake3")
	MaxFileSize    int64    `json:"max_file_size"`   // Skip files larger than this size in bytes (0 = no limit)
	IncludeHidden  bool     `json:"include_hidden"`  // Include files starting with . (default: false)
	LockfilesOnly  bool     `json:"lockfiles_only"`  // Only process lockfiles (default: false)
	IncludePattern []string `json:"include_pattern"` // Only hash files matching these patterns
	ExcludePattern []string `json:"exclude_pattern"` // Skip files matching these patterns
}

// NewProjectCalculator creates a digest calculator for the specified root directory.
// It loads .mitlignore patterns and configures the calculator according to options.
func NewProjectCalculator(root string, options Options) *ProjectCalculator {
	// Set default algorithm if not specified
	if options.Algorithm == "" {
		options.Algorithm = "blake3"
	}

	// Load .mitlignore patterns from the project directory
	ignoreMatcher, err := LoadIgnoreRulesFromProject(root)
	if err != nil {
		// If loading fails, create empty ignore rules
		ignoreMatcher = NewIgnoreRules()
	}

	// Create internal calculator with appropriate algorithm
	var algorithm HashAlgorithm
	switch options.Algorithm {
	case "blake3":
		algorithm = Blake3
	case "sha256":
		algorithm = SHA256
	default:
		algorithm = Blake3 // Default to Blake3 for performance
	}

	// Configure internal calculator
	calcOpts := CalculatorOptions{
		Algorithm:   algorithm,
		Parallel:    true,
		MaxWorkers:  4,
		Normalizer:  NewNormalizer(),
		IgnoreRules: ignoreMatcher,
	}

	internalCalc := NewCalculatorWithOptions(calcOpts)

	return &ProjectCalculator{
		root:          root,
		ignoreMatcher: ignoreMatcher,
		options:       options,
		internalCalc:  internalCalc,
	}
}

// Calculate computes the digest for the project directory.
// Returns a complete Digest struct with file metadata and hash information.
func (c *ProjectCalculator) Calculate(ctx context.Context) (*Digest, error) {
	// Validate options
	if err := c.options.Validate(); err != nil {
		return nil, err
	}
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// Run calculation in a goroutine so we can respect context cancellation reliably
	type res struct {
		dr  *CalcResult
		err error
	}
	ch := make(chan res, 1)
	go func() {
		r, e := c.internalCalc.CalculateDirectory(ctx, c.root)
		ch <- res{dr: r, err: e}
	}()

	var result *CalcResult
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case rr := <-ch:
		if rr.err != nil {
			return nil, fmt.Errorf("failed to calculate digest: %w", rr.err)
		}
		result = rr.dr
	}

	// Filter files based on options
	filteredFiles := c.filterFiles(result.Files)

	// Convert internal CalcFileInfo to our FileDigest format
	files := make([]FileDigest, 0, len(filteredFiles))
	for _, f := range filteredFiles {
		if f.Error != nil {
			continue // Skip files with errors
		}

		files = append(files, FileDigest{
			Path: f.Path,
			Hash: f.Hash,
			Size: f.Size,
		})
	}

	// Sort files for deterministic ordering
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Calculate final combined hash
	finalHash := c.calculateCombinedHash(files)

	return &Digest{
		Hash:      finalHash,
		Algorithm: c.options.Algorithm,
		Timestamp: time.Now().UTC(),
		FileCount: len(files),
		TotalSize: result.TotalSize,
		Files:     files,
		Options:   c.options,
	}, nil
}

// filterFiles applies the configured file filters to the result set.
func (c *ProjectCalculator) filterFiles(files []CalcFileInfo) []CalcFileInfo {
	if c.options.LockfilesOnly {
		return c.filterLockfilesOnly(files)
	}

	filtered := make([]CalcFileInfo, 0, len(files))
	for _, file := range files {
		if c.shouldIncludeFile(file) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// filterLockfilesOnly returns only lockfiles from the file list.
func (c *ProjectCalculator) filterLockfilesOnly(files []CalcFileInfo) []CalcFileInfo {
	// Known lockfile patterns
	lockfileNames := map[string]bool{
		"composer.lock":     true,
		"package-lock.json": true,
		"pnpm-lock.yaml":    true,
		"yarn.lock":         true,
		"go.sum":            true,
		"go.mod":            true,
		"Gemfile.lock":      true,
		"requirements.txt":  true,
		"poetry.lock":       true,
		"Pipfile.lock":      true,
		"Cargo.lock":        true,
	}

	filtered := make([]CalcFileInfo, 0)
	for _, file := range files {
		filename := filepath.Base(file.Path)
		if lockfileNames[filename] {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// shouldIncludeFile determines if a file should be included based on filters.
func (c *ProjectCalculator) shouldIncludeFile(file CalcFileInfo) bool {
	// Check file size limit
	if c.options.MaxFileSize > 0 && file.Size > c.options.MaxFileSize {
		return false
	}

	// Check hidden file setting
	filename := filepath.Base(file.Path)
	if !c.options.IncludeHidden && strings.HasPrefix(filename, ".") {
		return false
	}

	// Check extension filters
	if !c.matchesExtensionFilter(file.Path) {
		return false
	}

	return true
}

// matchesExtensionFilter checks if file should be included based on extension filters.
func (c *ProjectCalculator) matchesExtensionFilter(path string) bool {
	// If IncludePattern is set, file must match one of them
	if len(c.options.IncludePattern) > 0 {
		found := false
		for _, pattern := range c.options.IncludePattern {
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check ExcludePattern
	for _, pattern := range c.options.ExcludePattern {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return false
		}
	}

	return true
}

// calculateCombinedHash creates the final digest from all file hashes.
func (c *ProjectCalculator) calculateCombinedHash(files []FileDigest) string {
	hasher := sha256.New()

	// Include each file's path, size, and content hash for comprehensive digest
	for _, file := range files {
		// Format: path + size + hash + newline for deterministic ordering
		fmt.Fprintf(hasher, "%s\n%d\n%s\n", file.Path, file.Size, file.Hash)
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// Validate validates the options configuration.
func (o *Options) Validate() error {
	switch o.Algorithm {
	case "", "blake3", "sha256":
		// Valid algorithms
	default:
		return fmt.Errorf("unsupported algorithm: %s", o.Algorithm)
	}
	return nil
}

// formatBytes formats byte counts in human readable format.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// (Legacy helpers removed; use ProjectTag and the digest CLI instead.)

// ProjectTag computes the project digest and returns a short (12-char) tag
// suitable for container image tagging.
func ProjectTag(root string, options Options) (string, error) {
	if err := options.Validate(); err != nil {
		return "", err
	}
	calc := NewProjectCalculator(root, options)
	d, err := calc.Calculate(context.Background())
	if err != nil {
		return "", err
	}
	if len(d.Hash) < 12 {
		return "", fmt.Errorf("digest too short")
	}
	return d.Hash[:12], nil
}
