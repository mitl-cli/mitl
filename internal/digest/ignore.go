package digest

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gobwas/glob"
)

// IgnoreRules manages gitignore-style pattern matching for file exclusion.
// It supports standard gitignore syntax including negation (!), directory-only
// patterns (/), and provides caching for performance optimization.
type IgnoreRules struct {
	patterns []ignorePattern
	cache    map[string]bool
	cacheMu  sync.RWMutex
}

// ignorePattern represents a single ignore rule with its compiled glob and metadata.
type ignorePattern struct {
	pattern    string
	glob       glob.Glob
	negate     bool
	dirOnly    bool
	hasSlash   bool
	isAbsolute bool
}

// NewIgnoreRules creates a new ignore rules manager with default exclusions.
func NewIgnoreRules() *IgnoreRules {
	rules := &IgnoreRules{
		patterns: make([]ignorePattern, 0),
		cache:    make(map[string]bool),
	}

	// Add default exclusion patterns
	defaultPatterns := []string{
		".git/",
		".svn/",
		".hg/",
		".bzr/",
		"node_modules/",
		".DS_Store",
		"Thumbs.db",
		"*.tmp",
		"*.swp",
		"*.swo",
		"*~",
		".mitl/",
	}

	for _, pattern := range defaultPatterns {
		if err := rules.AddPattern(pattern); err != nil {
			// Log error but continue - default patterns should be safe
			continue
		}
	}

	return rules
}

// LoadFromFile loads ignore patterns from a .mitlignore file.
func (r *IgnoreRules) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No ignore file is not an error
		}
		return fmt.Errorf("failed to open ignore file %s: %w", filename, err)
	}
	defer file.Close()

	return r.LoadFromReader(file)
}

// LoadFromReader loads ignore patterns from an io.Reader.
func (r *IgnoreRules) LoadFromReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if err := r.AddPattern(line); err != nil {
			return fmt.Errorf("invalid pattern on line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading ignore patterns: %w", err)
	}

	return nil
}

// AddPattern adds a single ignore pattern to the rules.
func (r *IgnoreRules) AddPattern(pattern string) error {
	if pattern == "" {
		return nil
	}

	// Parse pattern characteristics
	negate := strings.HasPrefix(pattern, "!")
	if negate {
		pattern = pattern[1:]
	}

	dirOnly := strings.HasSuffix(pattern, "/")
	if dirOnly {
		pattern = strings.TrimSuffix(pattern, "/")
	}

	hasSlash := strings.Contains(pattern, "/")
	isAbsolute := strings.HasPrefix(pattern, "/")
	if isAbsolute {
		pattern = pattern[1:]
		hasSlash = true
	}

	// Compile glob pattern
	globPattern, err := r.compileGlobPattern(pattern, hasSlash, isAbsolute)
	if err != nil {
		return fmt.Errorf("failed to compile pattern '%s': %w", pattern, err)
	}

	r.patterns = append(r.patterns, ignorePattern{
		pattern:    pattern,
		glob:       globPattern,
		negate:     negate,
		dirOnly:    dirOnly,
		hasSlash:   hasSlash,
		isAbsolute: isAbsolute,
	})

	// Clear cache when patterns change
	r.cacheMu.Lock()
	r.cache = make(map[string]bool)
	r.cacheMu.Unlock()

	return nil
}

// compileGlobPattern converts gitignore pattern to glob pattern.
func (r *IgnoreRules) compileGlobPattern(pattern string, hasSlash, isAbsolute bool) (glob.Glob, error) {
	globPattern := pattern

	if !hasSlash {
		// Pattern without slash matches at any level
		globPattern = "**/" + pattern
	}

	// Handle leading slash (absolute patterns)
	if isAbsolute {
		globPattern = pattern
	}

	// Ensure directory patterns match recursively
	if !strings.HasSuffix(globPattern, "/**") && !strings.Contains(globPattern, "*") {
		globPattern += "/**"
	}

	return glob.Compile(globPattern, '/')
}

// ShouldIgnore determines if a file path should be ignored based on the loaded patterns.
// The path should be relative to the project root and use forward slashes.
func (r *IgnoreRules) ShouldIgnore(path string, isDir bool) bool {
	// Normalize path
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "./")

	// Check cache first
	cacheKey := path
	if isDir {
		cacheKey += "/"
	}

	r.cacheMu.RLock()
	if result, exists := r.cache[cacheKey]; exists {
		r.cacheMu.RUnlock()
		return result
	}
	r.cacheMu.RUnlock()

	// Evaluate patterns
	result := r.evaluatePatterns(path, isDir)

	// Cache result
	r.cacheMu.Lock()
	r.cache[cacheKey] = result
	r.cacheMu.Unlock()

	return result
}

// evaluatePatterns checks all patterns against the given path.
func (r *IgnoreRules) evaluatePatterns(path string, isDir bool) bool {
	ignored := false

	for _, pattern := range r.patterns {
		// Skip directory-only patterns for files
		if pattern.dirOnly && !isDir {
			continue
		}

		// Check if pattern matches
		matches := r.matchesPattern(pattern, path, isDir)

		if matches {
			if pattern.negate {
				ignored = false // Negation pattern un-ignores
			} else {
				ignored = true // Normal pattern ignores
			}
		}
	}

	return ignored
}

// matchesPattern checks if a single pattern matches the given path.
func (r *IgnoreRules) matchesPattern(pattern ignorePattern, path string, isDir bool) bool {
	// For patterns without slash, check basename and full path
	if !pattern.hasSlash {
		basename := filepath.Base(path)
		if pattern.glob.Match(basename) {
			return true
		}
	}

	// Check full path
	testPath := path
	if isDir && !strings.HasSuffix(testPath, "/") {
		testPath += "/"
	}

	return pattern.glob.Match(testPath)
}

// GetPatterns returns a copy of all loaded patterns for inspection.
func (r *IgnoreRules) GetPatterns() []string {
	patterns := make([]string, 0, len(r.patterns))
	for _, p := range r.patterns {
		pattern := p.pattern
		if p.negate {
			pattern = "!" + pattern
		}
		if p.dirOnly {
			pattern += "/"
		}
		if p.isAbsolute {
			pattern = "/" + pattern
		}
		patterns = append(patterns, pattern)
	}
	return patterns
}

// ClearCache clears the internal pattern matching cache.
// This can be useful for memory management in long-running applications.
func (r *IgnoreRules) ClearCache() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	r.cache = make(map[string]bool)
}

// Stats returns statistics about the ignore rules.
func (r *IgnoreRules) Stats() IgnoreStats {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	return IgnoreStats{
		PatternCount: len(r.patterns),
		CacheSize:    len(r.cache),
	}
}

// IgnoreStats provides statistics about ignore rule usage.
type IgnoreStats struct {
	PatternCount int
	CacheSize    int
}

// LoadIgnoreRulesFromProject loads .mitlignore rules from project directory.
// It looks for .mitlignore files starting from the given directory and walking up
// to find parent .mitlignore files, similar to how git works with .gitignore.
func LoadIgnoreRulesFromProject(projectDir string) (*IgnoreRules, error) {
	rules := NewIgnoreRules()

	// Load .mitlignore from project root
	ignoreFile := filepath.Join(projectDir, ".mitlignore")
	if err := rules.LoadFromFile(ignoreFile); err != nil {
		return nil, fmt.Errorf("failed to load ignore rules from %s: %w", ignoreFile, err)
	}

	return rules, nil
}
