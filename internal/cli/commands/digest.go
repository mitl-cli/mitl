// Package commands provides the digest command implementation for CLI access.
// This command allows users to calculate, inspect, and debug project digests
// for cache validation and troubleshooting purposes.

package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mitl/internal/digest"
)

// DigestCommand provides functionality to calculate and inspect project digests.
// It supports various options for debugging cache issues and understanding
// what files affect the project digest calculation.
type DigestCommand struct{}

// NewDigestCommand creates a new digest command instance.
func NewDigestCommand() *DigestCommand {
	return &DigestCommand{}
}

// Run executes the digest command with the provided arguments.
// Supports flags: --verbose, --files, --save, --compare, --lockfiles-only
func (d *DigestCommand) Run(args []string) error {
	// Parse command line flags
	config := d.parseFlags(args)

	if config.showHelp {
		d.showHelp()
		return nil
	}

	// Handle lockfiles-only mode
	if config.lockfilesOnly {
        return d.runLockfilesMode(&config)
	}

	// Calculate project digest
	calculator := digest.NewProjectCalculator(config.rootDir, config.options)
	projectDigest, err := calculator.Calculate(context.Background())
	if err != nil {
		return fmt.Errorf("failed to calculate digest: %w", err)
	}

	// Display results
    d.displayResults(projectDigest, &config)

	// Handle comparison if requested
	if config.comparePath != "" {
        if err := d.runComparison(projectDigest, config.comparePath, &config); err != nil {
            return fmt.Errorf("comparison failed: %w", err)
        }
	}

	// Save digest if requested
    if config.savePath != "" {
        if err := digest.SaveDigest(projectDigest, config.savePath); err != nil {
            return fmt.Errorf("failed to save digest: %w", err)
        }
        fmt.Printf("💾 Digest saved to: %s\n", config.savePath)
    }

	return nil
}

// digestConfig holds the parsed command configuration.
type digestConfig struct {
	rootDir       string
	verbose       bool
	showFiles     bool
	savePath      string
	comparePath   string
	lockfilesOnly bool
	showHelp      bool
	options       digest.Options
}

// parseFlags parses command line arguments and returns configuration.
func (d *DigestCommand) parseFlags(args []string) digestConfig {
	config := digestConfig{
		rootDir: ".",
		options: digest.Options{
			Algorithm: "sha256", // default
		},
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			config.showHelp = true
		case "-v", "--verbose":
			config.verbose = true
		case "--files":
			config.showFiles = true
		case "--save":
			if i+1 < len(args) {
				config.savePath = args[i+1]
				i++
			}
		case "--compare":
			if i+1 < len(args) {
				config.comparePath = args[i+1]
				i++
			}
		case "--lockfiles-only":
			config.lockfilesOnly = true
		case "--algorithm":
			if i+1 < len(args) {
				config.options.Algorithm = args[i+1]
				i++
			}
		case "--max-size":
			if i+1 < len(args) {
				if size, err := strconv.ParseInt(args[i+1], 10, 64); err == nil {
					config.options.MaxFileSize = size
				}
				i++
			}
		case "--include-hidden":
			config.options.IncludeHidden = true
		case "--only-ext":
			if i+1 < len(args) {
				exts := strings.Split(args[i+1], ",")
				// Convert extensions to glob patterns
				patterns := make([]string, 0, len(exts))
				for _, e := range exts {
					e = strings.TrimSpace(e)
					if e == "" {
						continue
					}
					if strings.ContainsAny(e, "*?[") {
						patterns = append(patterns, e)
					} else if strings.HasPrefix(e, ".") {
						patterns = append(patterns, "*"+e)
					} else {
						patterns = append(patterns, "*."+e)
					}
				}
				config.options.IncludePattern = patterns
				i++
			}
		case "--exclude-ext":
			if i+1 < len(args) {
				exts := strings.Split(args[i+1], ",")
				patterns := make([]string, 0, len(exts))
				for _, e := range exts {
					e = strings.TrimSpace(e)
					if e == "" {
						continue
					}
					if strings.ContainsAny(e, "*?[") {
						patterns = append(patterns, e)
					} else if strings.HasPrefix(e, ".") {
						patterns = append(patterns, "*"+e)
					} else {
						patterns = append(patterns, "*."+e)
					}
				}
				config.options.ExcludePattern = patterns
				i++
			}
		case "--root":
			if i+1 < len(args) {
				config.rootDir = args[i+1]
				i++
			}
		}
	}

	return config
}

// displayResults shows the digest calculation results.
func (d *DigestCommand) displayResults(projectDigest *digest.Digest, config *digestConfig) {
	// Main digest information
	fmt.Printf("🔐 Digest: %s\n", projectDigest.Hash[:16])
	fmt.Printf("📁 Files: %d\n", projectDigest.FileCount)

    if config.verbose {
		fmt.Printf("Algorithm: %s\n", projectDigest.Algorithm)
		fmt.Printf("Full Hash: %s\n", projectDigest.Hash)
		fmt.Printf("Timestamp: %s\n", projectDigest.Timestamp.Format(time.RFC3339))
		fmt.Printf("Total Size: %s\n", d.formatFileSize(projectDigest.TotalSize))

		// Show file details if requested
        if config.showFiles {
            d.displayFileList(projectDigest.Files)
        }
    }
}

// displayFileList shows detailed information about files included in the digest.
func (d *DigestCommand) displayFileList(files []digest.FileDigest) {
	fmt.Println("\nFiles included in digest:")

	for _, f := range files {
		sizeStr := d.formatFileSize(f.Size)
		fmt.Printf("  %s (%s) - %s\n", f.Path, sizeStr, f.Hash[:12])
	}
}

// formatFileSize formats a file size in bytes to a human-readable string.
func (d *DigestCommand) formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// runLockfilesMode handles the --lockfiles-only flag.
func (d *DigestCommand) runLockfilesMode(config *digestConfig) error {
	fmt.Println("🔐 Lockfiles Digest")

	hasher := digest.NewLockfileHasher(config.rootDir)
	lockfilesHash, err := hasher.HashLockfiles()
	if err != nil {
		return fmt.Errorf("failed to calculate lockfiles hash: %w", err)
	}

	fmt.Printf("Hash: %s\n", lockfilesHash)

    if config.verbose {
		// Show which lockfiles were found
		lockfileNames := []string{
			"composer.lock", "package-lock.json", "pnpm-lock.yaml", "yarn.lock",
			"go.sum", "go.mod", "Gemfile.lock", "requirements.txt", "poetry.lock",
			"Pipfile.lock", "cargo.lock",
		}

		fmt.Println("\nLockfiles found:")
		found := false
		for _, name := range lockfileNames {
            path := filepath.Join(config.rootDir, name)
			if _, err := os.Stat(path); err == nil {
				fmt.Printf("  ✓ %s\n", name)
				found = true
			}
		}

		if !found {
			fmt.Println("  (none)")
		}
	}

	return nil
}

// runComparison compares the current digest with a saved one.
func (d *DigestCommand) runComparison(newDigest *digest.Digest, comparePath string, config *digestConfig) error {
	fmt.Printf("\n🔍 Comparing with saved digest: %s\n", comparePath)

	comparison, err := digest.CompareWithSaved(comparePath, newDigest)
	if err != nil {
		return fmt.Errorf("failed to load and compare digest: %w", err)
	}

	// Show comparison summary
	fmt.Printf("Status: %s\n", comparison.Summary())

	if config.verbose && !comparison.Identical {
		// Show detailed changes
		if len(comparison.Modified) > 0 {
			fmt.Printf("\nChanged files (%d):\n", len(comparison.Modified))
			for _, file := range comparison.Modified {
				fmt.Printf("  📝 %s\n", file)
			}
		}

		if len(comparison.Added) > 0 {
			fmt.Printf("\nAdded files (%d):\n", len(comparison.Added))
			for _, file := range comparison.Added {
				fmt.Printf("  ➕ %s\n", file)
			}
		}

		if len(comparison.Removed) > 0 {
			fmt.Printf("\nRemoved files (%d):\n", len(comparison.Removed))
			for _, file := range comparison.Removed {
				fmt.Printf("  ➖ %s\n", file)
			}
		}
	}

	return nil
}

// showHelp displays the command usage information.
func (d *DigestCommand) showHelp() {
	fmt.Println(`mitl digest - Calculate and inspect project digests

USAGE:
    mitl digest [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -v, --verbose           Show detailed information
    --files                 List all files included in digest
    --save PATH             Save digest to file for future comparison
    --compare PATH          Compare current digest with saved digest
    --lockfiles-only        Calculate digest of lockfiles only
    --algorithm ALGO        Hash algorithm: sha256 (default), blake3
    --max-size BYTES        Skip files larger than specified size
    --include-hidden        Include hidden files (starting with .)
    --only-ext EXTS         Only include files with specified extensions (comma-separated)
    --exclude-ext EXTS      Exclude files with specified extensions (comma-separated)
    --root DIR              Project root directory (default: current directory)

EXAMPLES:
    mitl digest                                    # Calculate basic digest
    mitl digest --verbose --files                  # Show detailed file list
    mitl digest --save .mitl/digest.json          # Save digest for later comparison
    mitl digest --compare .mitl/digest.json       # Compare with saved digest
    mitl digest --lockfiles-only                  # Hash only dependency lockfiles
    mitl digest --algorithm blake3 --verbose      # Use Blake3 algorithm
    mitl digest --only-ext .go,.mod --verbose     # Only hash Go files

The digest command helps debug cache issues by showing exactly what files
affect your project's cache key and how changes impact the digest.`)
}

// Digest function provides the main entry point for the digest command.
// This maintains compatibility with the existing CLI structure.
func Digest(args []string) error {
	cmd := NewDigestCommand()
	return cmd.Run(args)
}
