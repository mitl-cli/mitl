package digest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

// Calculator provides enhanced digest calculation with deterministic processing,
// content normalization, and ignore rule support. It uses Blake3 hashing by default
// for better performance while maintaining backwards compatibility with SHA256.
type Calculator struct {
	normalizer  *Normalizer
	ignoreRules *IgnoreRules
	algorithm   HashAlgorithm
	parallel    bool
	maxWorkers  int
	bufferPool  sync.Pool
}

// HashAlgorithm defines the supported hashing algorithms.
type HashAlgorithm int

const (
	// SHA256 provides backwards compatibility with existing digests
	SHA256 HashAlgorithm = iota
	// Blake3 provides better performance for new digests
	Blake3
)

// CalculatorOptions configures the calculator behavior.
type CalculatorOptions struct {
	Algorithm   HashAlgorithm
	Parallel    bool
	MaxWorkers  int
	Normalizer  *Normalizer
	IgnoreRules *IgnoreRules
}

// workItem is an internal unit of work for hashing
type workItem struct {
	path  string
	index int
}

// CalcFileInfo contains metadata about a processed file.
type CalcFileInfo struct {
	Path         string
	Size         int64
	Hash         string
	IsNormalized bool
	Error        error
}

// CalcResult contains the complete digest calculation results.
type CalcResult struct {
	Algorithm    HashAlgorithm
	Digest       string
	Files        []CalcFileInfo
	TotalFiles   int
	TotalSize    int64
	IgnoredFiles int
	Errors       []error
}

// NewCalculator creates a new digest calculator with default settings.
func NewCalculator() *Calculator {
	return NewCalculatorWithOptions(CalculatorOptions{
		Algorithm:   Blake3,
		Parallel:    true,
		MaxWorkers:  4,
		Normalizer:  NewNormalizer(),
		IgnoreRules: NewIgnoreRules(),
	})
}

// NewCalculatorWithOptions creates a calculator with custom configuration.
func NewCalculatorWithOptions(opts CalculatorOptions) *Calculator {
	if opts.MaxWorkers <= 0 {
		opts.MaxWorkers = 4
	}
	if opts.Normalizer == nil {
		opts.Normalizer = NewNormalizer()
	}
	if opts.IgnoreRules == nil {
		opts.IgnoreRules = NewIgnoreRules()
	}

	return &Calculator{
		normalizer:  opts.Normalizer,
		ignoreRules: opts.IgnoreRules,
		algorithm:   opts.Algorithm,
		parallel:    opts.Parallel,
		maxWorkers:  opts.MaxWorkers,
		bufferPool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 32*1024) // 32KB buffer
				return &buf
			},
		},
	}
}

// CalculateDirectory computes digest for all files in a directory tree.
// Files are processed in deterministic order and filtered by ignore rules.
func (c *Calculator) CalculateDirectory(ctx context.Context, rootDir string) (*CalcResult, error) {
	// Collect all files first for deterministic processing
	files, err := c.collectFiles(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to collect files: %w", err)
	}

	// Sort files for deterministic order
	sort.Strings(files)

	// Process files
	return c.processFiles(ctx, rootDir, files)
}

// CalculateFiles computes digest for specific files.
func (c *Calculator) CalculateFiles(ctx context.Context, files []string) (*CalcResult, error) {
	// Sort for deterministic order
	sortedFiles := make([]string, len(files))
	copy(sortedFiles, files)
	sort.Strings(sortedFiles)

	return c.processFiles(ctx, "", sortedFiles)
}

// CalculateFile computes digest for a single file with normalization.
func (c *Calculator) CalculateFile(filePath string) (string, error) {
	// Backwards-compatible wrapper without context
	return c.calculateFileWithContext(context.Background(), filePath)
}

// calculateFileWithContext reads file content in chunks, allowing context cancellation.
func (c *Calculator) calculateFileWithContext(ctx context.Context, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Read file content in chunks to allow cancellation checks
	var buf []byte
	chunk := make([]byte, 32*1024)
	for {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
			}
			if d, ok := ctx.Deadline(); ok {
				// If deadline is extremely soon, yield briefly to allow cancellation to propagate
				if rem := time.Until(d); rem > 0 && rem < 2*time.Millisecond {
					time.Sleep(rem)
				}
			}
		}
		n, rErr := file.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
		}
		if rErr == io.EOF {
			break
		}
		if rErr != nil {
			return "", fmt.Errorf("failed to read file %s: %w", filePath, rErr)
		}
	}

	// Apply normalization
	normalized, err := c.normalizer.Normalize(buf)
	if err != nil {
		return "", fmt.Errorf("failed to normalize file %s: %w", filePath, err)
	}
	if ctx != nil && ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Calculate hash
	return c.hashContent(normalized), nil
}

// collectFiles walks the directory tree and returns all file paths.
func (c *Calculator) collectFiles(rootDir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Convert to relative path
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Apply ignore rules
		if c.ignoreRules.ShouldIgnore(relPath, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Only include regular files
		if d.Type().IsRegular() {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// processFiles processes a list of files and computes the combined digest.
func (c *Calculator) processFiles(ctx context.Context, rootDir string, files []string) (*CalcResult, error) {
	result := &CalcResult{
		Algorithm: c.algorithm,
		Files:     make([]CalcFileInfo, 0, len(files)),
		Errors:    make([]error, 0),
	}

	// Create work channel and result channel
	workCh := make(chan workItem, len(files))
	resultCh := make(chan CalcFileInfo, len(files))

	// Start workers if parallel processing is enabled
	var wg sync.WaitGroup
	workers := 1
	if c.parallel {
		workers = minInt(c.maxWorkers, len(files))
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.worker(ctx, rootDir, workCh, resultCh)
		}()
	}

	// Send work items
	go func() {
		defer close(workCh)
		for i, path := range files {
			select {
			case workCh <- workItem{path: path, index: i}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Process results maintaining order
	fileResults := make([]CalcFileInfo, len(files))
	for fileInfo := range resultCh {
		if fileInfo.Error != nil {
			result.Errors = append(result.Errors, fileInfo.Error)
		} else {
			fileResults[len(result.Files)] = fileInfo
			result.Files = append(result.Files, fileInfo)
			result.TotalSize += fileInfo.Size
		}
	}

	result.TotalFiles = len(result.Files)

	// Check for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Calculate combined digest
	result.Digest = c.calculateCombinedDigest(result.Files)

	return result, nil
}

// worker processes files from the work channel.
func (c *Calculator) worker(ctx context.Context, rootDir string, workCh <-chan workItem, resultCh chan<- CalcFileInfo) {
	for work := range workCh {
		select {
		case <-ctx.Done():
			return
		default:
		}

		fileInfo := c.processFileWithContext(ctx, work.path, rootDir)
		resultCh <- fileInfo
	}
}

// processFile processes a single file and returns its information.
func (c *Calculator) processFile(filePath, rootDir string) CalcFileInfo { // retained for compatibility
	return c.processFileWithContext(context.Background(), filePath, rootDir)
}

func (c *Calculator) processFileWithContext(ctx context.Context, filePath, rootDir string) CalcFileInfo {
	// Get relative path for result
	relPath := filePath
	if rootDir != "" {
		if rel, err := filepath.Rel(rootDir, filePath); err == nil {
			relPath = rel
		}
	}

	fileInfo := CalcFileInfo{
		Path: relPath,
	}

	// Get file stats
	stat, err := os.Stat(filePath)
	if err != nil {
		fileInfo.Error = fmt.Errorf("failed to stat file %s: %w", filePath, err)
		return fileInfo
	}
	fileInfo.Size = stat.Size()

	// Calculate hash with context support
	hash, err := c.calculateFileWithContext(ctx, filePath)
	if err != nil {
		fileInfo.Error = fmt.Errorf("failed to hash file %s: %w", filePath, err)
		return fileInfo
	}

	fileInfo.Hash = hash
	fileInfo.IsNormalized = true

	return fileInfo
}

// calculateCombinedDigest creates a combined hash from all file hashes.
func (c *Calculator) calculateCombinedDigest(files []CalcFileInfo) string {
	// Sort files by path for deterministic order
	sortedFiles := make([]CalcFileInfo, len(files))
	copy(sortedFiles, files)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].Path < sortedFiles[j].Path
	})

	// Create combined input for hashing
	var input strings.Builder
	for _, file := range sortedFiles {
		// Include both path and hash for comprehensive digest
		input.WriteString(file.Path)
		input.WriteString(":")
		input.WriteString(file.Hash)
		input.WriteString("\n")
	}

	return c.hashContent([]byte(input.String()))
}

// hashContent calculates hash of content using the configured algorithm.
func (c *Calculator) hashContent(content []byte) string {
	switch c.algorithm {
	case SHA256:
		hash := sha256.Sum256(content)
		return hex.EncodeToString(hash[:])
	case Blake3:
		hash := blake3.Sum256(content)
		return hex.EncodeToString(hash[:])
	default:
		// Fallback to SHA256
		hash := sha256.Sum256(content)
		return hex.EncodeToString(hash[:])
	}
}

// minInt returns the minimum of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
