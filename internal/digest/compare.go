// Package digest comparison functionality for determining when rebuilds are needed.
// This module provides tools to compare digests and identify what changed between
// two project states, enabling intelligent cache invalidation decisions.

package digest

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Comparison represents the result of comparing two digests.
// It provides detailed information about what changed between the old and new state.
type Comparison struct {
	Old       *Digest  `json:"old"`       // Previous digest state
	New       *Digest  `json:"new"`       // Current digest state
	Identical bool     `json:"identical"` // Whether digests are identical
	Added     []string `json:"added"`     // Files that were added
	Modified  []string `json:"modified"`  // Files that were modified (content changed)
	Removed   []string `json:"removed"`   // Files that were removed
	Reason    string   `json:"reason"`    // Human-readable explanation of changes
}

// Compare compares two digests and returns detailed information about differences.
// This is the primary function for determining if a rebuild is needed and why.
func Compare(old, new *Digest) *Comparison {
	comp := &Comparison{
		Old: old,
		New: new,
	}

	// Quick hash comparison for overall change detection
	if old.Hash == new.Hash {
		comp.Identical = true
	} else {
		comp.Identical = false
		comp.findDifferences()
	}

	return comp
}

// findDifferences analyzes the file lists to determine what specifically changed.
// This provides transparency about why a cache miss occurred.
func (c *Comparison) findDifferences() {
	oldFiles := make(map[string]string)
	newFiles := make(map[string]string)

	// Build maps for efficient lookup
	for _, f := range c.Old.Files {
		oldFiles[f.Path] = f.Hash
	}

	for _, f := range c.New.Files {
		newFiles[f.Path] = f.Hash
	}

	// Find changed and removed files
	for path, oldHash := range oldFiles {
		if newHash, exists := newFiles[path]; exists {
			if oldHash != newHash {
				c.Modified = append(c.Modified, path)
			}
		} else {
			c.Removed = append(c.Removed, path)
		}
	}

	// Find added files
	for path := range newFiles {
		if _, exists := oldFiles[path]; !exists {
			c.Added = append(c.Added, path)
		}
	}

	// Sort results for consistent output
	sort.Strings(c.Modified)
	sort.Strings(c.Added)
	sort.Strings(c.Removed)

	// Generate human-readable reason
	c.Reason = c.generateReason()
}

// generateReason creates a human-readable explanation of what changed.
func (c *Comparison) generateReason() string {
	var reasons []string

	if len(c.Modified) > 0 {
		reasons = append(reasons, fmt.Sprintf("%d files changed", len(c.Modified)))
	}
	if len(c.Added) > 0 {
		reasons = append(reasons, fmt.Sprintf("%d files added", len(c.Added)))
	}
	if len(c.Removed) > 0 {
		reasons = append(reasons, fmt.Sprintf("%d files removed", len(c.Removed)))
	}

	if len(reasons) == 0 {
		return "digest algorithm or metadata changed"
	}

	if len(reasons) == 1 {
		return reasons[0]
	}

	// Join with commas and "and" for the last item
	last := reasons[len(reasons)-1]
	others := reasons[:len(reasons)-1]
	return fmt.Sprintf("%s and %s", joinWithCommas(others), last)
}

// joinWithCommas joins a slice of strings with commas.
func joinWithCommas(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}

	result := ""
	for i, item := range items {
		if i > 0 {
			result += ", "
		}
		result += item
	}
	return result
}

// HasSignificantChanges determines if the changes should trigger a rebuild.
// Some changes might be cosmetic (like .mitlignore updates) and may not require rebuilds.
func (c *Comparison) HasSignificantChanges() bool {
	// Any file content changes are significant
	if len(c.Modified) > 0 || len(c.Added) > 0 || len(c.Removed) > 0 {
		return true
	}

	// Algorithm changes are significant
	if c.Old.Algorithm != c.New.Algorithm {
		return true
	}

	return false
}

// GetAffectedFiles returns all files that were part of the change.
// This can be useful for incremental processing or debugging.
func (c *Comparison) GetAffectedFiles() []string {
	affected := make([]string, 0, len(c.Modified)+len(c.Added)+len(c.Removed))
	affected = append(affected, c.Modified...)
	affected = append(affected, c.Added...)
	affected = append(affected, c.Removed...)
	sort.Strings(affected)
	return affected
}

// SaveDigest saves a digest to a file for future comparison.
// The digest is stored as JSON with pretty formatting for readability.
func SaveDigest(digest *Digest, path string) error {
	if digest == nil {
		return fmt.Errorf("nil digest")
	}
	data, err := json.MarshalIndent(digest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal digest to JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write digest file: %w", err)
	}

	return nil
}

// LoadDigest loads a previously saved digest from a file.
// Returns an error if the file doesn't exist or contains invalid JSON.
func LoadDigest(path string) (*Digest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read digest file: %w", err)
	}

	var digest Digest
	if err := json.Unmarshal(data, &digest); err != nil {
		return nil, fmt.Errorf("failed to parse digest JSON: %w", err)
	}

	return &digest, nil
}

// CompareWithSaved loads a saved digest and compares it with a new one.
// This is a convenience function that combines LoadDigest and Compare.
func CompareWithSaved(savedPath string, newDigest *Digest) (*Comparison, error) {
	oldDigest, err := LoadDigest(savedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load saved digest: %w", err)
	}

	return Compare(oldDigest, newDigest), nil
}

// Summary provides a concise text summary of the comparison results.
// This is useful for CLI output and logging.
func (c *Comparison) Summary() string {
	if c.Identical {
		return "Digests are identical"
	}

	var parts []string
	if len(c.Added) > 0 {
		if len(c.Added) == 1 {
			parts = append(parts, "1 file added")
		} else {
			parts = append(parts, fmt.Sprintf("%d files added", len(c.Added)))
		}
	}
	if len(c.Modified) > 0 {
		if len(c.Modified) == 1 {
			parts = append(parts, "1 file modified")
		} else {
			parts = append(parts, fmt.Sprintf("%d files modified", len(c.Modified)))
		}
	}
	if len(c.Removed) > 0 {
		if len(c.Removed) == 1 {
			parts = append(parts, "1 file removed")
		} else {
			parts = append(parts, fmt.Sprintf("%d files removed", len(c.Removed)))
		}
	}

	if len(parts) == 0 {
		return "Digest algorithm or metadata changed"
	}
	if len(parts) == 1 {
		return parts[0]
	}
	if len(parts) == 2 {
		return parts[0] + ", " + parts[1]
	}
	// For 3 parts: "X files added, Y files modified, Z files removed"
	return parts[0] + ", " + parts[1] + ", " + parts[2]
}
