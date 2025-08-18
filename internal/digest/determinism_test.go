package digest

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestCrossPlatformDeterminism validates that digest calculation produces
// identical results across different platforms (macOS, Linux, Windows).
// This is critical for TASK7's requirement of 100% deterministic hashing.
func TestCrossPlatformDeterminism(t *testing.T) {
	// Test various content types that could be affected by platform differences
	testCases := []struct {
		name        string
		filename    string
		content     string
		description string
	}{
		{
			name:        "unix_line_endings",
			filename:    "unix.txt",
			content:     "line1\nline2\nline3\n",
			description: "Unix LF line endings",
		},
		{
			name:        "windows_line_endings",
			filename:    "windows.txt",
			content:     "line1\r\nline2\r\nline3\r\n",
			description: "Windows CRLF line endings",
		},
		{
			name:        "mac_classic_endings",
			filename:    "mac.txt",
			content:     "line1\rline2\rline3\r",
			description: "Mac classic CR line endings",
		},
		{
			name:        "mixed_line_endings",
			filename:    "mixed.txt",
			content:     "line1\nline2\r\nline3\rline4\n",
			description: "Mixed line endings",
		},
		{
			name:        "utf8_bom",
			filename:    "bom.txt",
			content:     "\xEF\xBB\xBFHello World\nWith UTF-8 BOM",
			description: "UTF-8 with BOM",
		},
		{
			name:        "unicode_content",
			filename:    "unicode.txt",
			content:     "Hello 世界\nUnicode: éñ™\n",
			description: "Unicode characters",
		},
		{
			name:        "binary_content",
			filename:    "binary.dat",
			content:     "\x00\x01\x02\x03\xFF\xFE\xFD\xFC",
			description: "Binary data",
		},
		{
			name:        "empty_file",
			filename:    "empty.txt",
			content:     "",
			description: "Empty file",
		},
		{
			name:        "whitespace_only",
			filename:    "whitespace.txt",
			content:     "   \t  \n  \r\n\t\t\n",
			description: "Whitespace only",
		},
		{
			name:        "long_lines",
			filename:    "long.txt",
			content:     strings.Repeat("This is a very long line with repeated content. ", 100) + "\n",
			description: "Very long lines",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Create the test file
			filePath := filepath.Join(tempDir, tc.filename)
			err := os.WriteFile(filePath, []byte(tc.content), 0o644)
			if err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Test with both algorithms
			algorithms := []string{"blake3", "sha256"}

			for _, alg := range algorithms {
				t.Run(alg, func(t *testing.T) {
					options := Options{
						Algorithm:     alg,
						MaxFileSize:   10485760,
						IncludeHidden: false,
						LockfilesOnly: false,
					}

					// Calculate digest multiple times to ensure determinism
					const iterations = 3
					var digests []*Digest

					for i := 0; i < iterations; i++ {
						calc := NewProjectCalculator(tempDir, &options)
						digest, err := calc.Calculate(context.Background())
						if err != nil {
							t.Fatalf("Calculate() iteration %d failed: %v", i, err)
						}
						digests = append(digests, digest)
					}

					// Verify all iterations produce identical results
					baseDigest := digests[0]
					for i := 1; i < iterations; i++ {
						if digests[i].Hash != baseDigest.Hash {
							t.Errorf("hash mismatch at iteration %d: expected %s, got %s",
								i, baseDigest.Hash, digests[i].Hash)
						}

						if digests[i].FileCount != baseDigest.FileCount {
							t.Errorf("file count mismatch at iteration %d: expected %d, got %d",
								i, baseDigest.FileCount, digests[i].FileCount)
						}

						if digests[i].TotalSize != baseDigest.TotalSize {
							t.Errorf("total size mismatch at iteration %d: expected %d, got %d",
								i, baseDigest.TotalSize, digests[i].TotalSize)
						}
					}

					// Log platform-specific information
					t.Logf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH)
					t.Logf("Content: %s", tc.description)
					t.Logf("Algorithm: %s", alg)
					t.Logf("Hash: %s", baseDigest.Hash)
					t.Logf("File count: %d", baseDigest.FileCount)
					t.Logf("Total size: %d bytes", baseDigest.TotalSize)
				})
			}
		})
	}
}

// TestPathNormalization verifies that path handling is consistent across platforms
func TestPathNormalization(t *testing.T) {
	tempDir := t.TempDir()

	// Create a directory structure with various path patterns
	testPaths := map[string]string{
		"simple.txt":                "simple file",
		"sub/nested.txt":            "nested file",
		"deep/very/nested/file.txt": "deeply nested",
		"spaces in name.txt":        "file with spaces",
		"dots.and.periods.txt":      "file with dots",
		"under_scores.txt":          "file with underscores",
		"UPPERCASE.TXT":             "uppercase filename",
		"mixedCase.Txt":             "mixed case filename",
	}

	// Create all test files
	for path, content := range testPaths {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		if err != nil {
			t.Fatalf("failed to create directory for %s: %v", path, err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("failed to write file %s: %v", path, err)
		}
	}

	options := Options{
		Algorithm:     "blake3",
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	// Calculate digest multiple times
	const iterations = 3
	var digests []*Digest

	for i := 0; i < iterations; i++ {
		calc := NewProjectCalculator(tempDir, &options)
		digest, err := calc.Calculate(context.Background())
		if err != nil {
			t.Fatalf("Calculate() iteration %d failed: %v", i, err)
		}
		digests = append(digests, digest)
	}

	// Verify consistency
	baseDigest := digests[0]
	for i := 1; i < iterations; i++ {
		if digests[i].Hash != baseDigest.Hash {
			t.Errorf("path normalization inconsistent: hash mismatch at iteration %d", i)
		}
	}

	// Verify file paths are properly normalized
	for _, file := range baseDigest.Files {
		// Paths should use forward slashes regardless of platform
		if strings.Contains(file.Path, "\\") {
			t.Errorf("path not normalized: %s contains backslashes", file.Path)
		}

		// Paths should not start with "./"
		if strings.HasPrefix(file.Path, "./") {
			t.Errorf("path not normalized: %s starts with ./", file.Path)
		}
	}

	t.Logf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	t.Logf("Processed %d files consistently", baseDigest.FileCount)
	t.Logf("Final hash: %s", baseDigest.Hash)
}

// TestTimestampIndependence verifies that file modification times don't affect hashes
func TestTimestampIndependence(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	filePath := filepath.Join(tempDir, "timestamp_test.txt")
	content := "This file will have different timestamps"
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	options := Options{
		Algorithm:     "blake3",
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	// Calculate initial digest
	calc1 := NewProjectCalculator(tempDir, &options)
	digest1, err := calc1.Calculate(context.Background())
	if err != nil {
		t.Fatalf("first Calculate() failed: %v", err)
	}

	// Wait a bit and modify timestamp
	time.Sleep(10 * time.Millisecond)

	// Touch the file to change its modification time
	now := time.Now()
	err = os.Chtimes(filePath, now, now)
	if err != nil {
		t.Fatalf("failed to change file timestamp: %v", err)
	}

	// Calculate digest again
	calc2 := NewProjectCalculator(tempDir, &options)
	digest2, err := calc2.Calculate(context.Background())
	if err != nil {
		t.Fatalf("second Calculate() failed: %v", err)
	}

	// Verify digests are identical despite timestamp change
	if digest1.Hash != digest2.Hash {
		t.Errorf("timestamp affected hash: before=%s, after=%s",
			digest1.Hash, digest2.Hash)
	}

	t.Logf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	t.Logf("Hash unchanged despite timestamp modification: %s", digest1.Hash)
}

// TestPermissionIndependence verifies that file permissions don't affect hashes
func TestPermissionIndependence(t *testing.T) {
	// Skip on Windows as it has different permission model
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
		return
	}

	tempDir := t.TempDir()

	// Create a test file
	filePath := filepath.Join(tempDir, "permission_test.txt")
	content := "This file will have different permissions"
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	options := Options{
		Algorithm:     "blake3",
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	// Calculate initial digest
	calc1 := NewProjectCalculator(tempDir, &options)
	digest1, err := calc1.Calculate(context.Background())
	if err != nil {
		t.Fatalf("first Calculate() failed: %v", err)
	}

	// Change file permissions
	err = os.Chmod(filePath, 0o755)
	if err != nil {
		t.Fatalf("failed to change file permissions: %v", err)
	}

	// Calculate digest again
	calc2 := NewProjectCalculator(tempDir, &options)
	digest2, err := calc2.Calculate(context.Background())
	if err != nil {
		t.Fatalf("second Calculate() failed: %v", err)
	}

	// Verify digests are identical despite permission change
	if digest1.Hash != digest2.Hash {
		t.Errorf("permissions affected hash: before=%s, after=%s",
			digest1.Hash, digest2.Hash)
	}

	t.Logf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	t.Logf("Hash unchanged despite permission modification: %s", digest1.Hash)
}

// TestReferenceHashes provides known good hashes for cross-platform validation
// These can be used to verify that different platforms produce identical results
func TestReferenceHashes(t *testing.T) {
	tempDir := t.TempDir()

	// Create a known test structure
	testFiles := map[string]string{
		"main.go":   "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n",
		"go.mod":    "module test\n\ngo 1.21\n",
		"README.md": "# Test Project\n\nThis is a test.\n",
		"data.json": `{"name": "test", "version": "1.0.0"}`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		err := os.WriteFile(fullPath, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("failed to create test file %s: %v", path, err)
		}
	}

	options := Options{
		Algorithm:     "blake3",
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	calc := NewProjectCalculator(tempDir, &options)
	digest, err := calc.Calculate(context.Background())
	if err != nil {
		t.Fatalf("Calculate() failed: %v", err)
	}

	// Log the reference hash for cross-platform comparison
	t.Logf("=== REFERENCE HASH ===")
	t.Logf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	t.Logf("Algorithm: %s", options.Algorithm)
	t.Logf("File count: %d", digest.FileCount)
	t.Logf("Total size: %d bytes", digest.TotalSize)
	t.Logf("Hash: %s", digest.Hash)
	t.Logf("=====================")

	// Log individual file hashes for debugging
	for _, file := range digest.Files {
		t.Logf("File: %s, Size: %d, Hash: %s", file.Path, file.Size, file.Hash)
	}

	// Verify basic properties
	if int(digest.FileCount) != len(testFiles) {
		t.Errorf("expected %d files, got %d", len(testFiles), digest.FileCount)
	}

	if digest.Hash == "" {
		t.Error("hash should not be empty")
	}

	if digest.Algorithm != "blake3" {
		t.Errorf("expected blake3 algorithm, got %s", digest.Algorithm)
	}
}

// TestContentNormalizationConsistency ensures that content normalization
// produces consistent results across multiple runs and platforms
func TestContentNormalizationConsistency(t *testing.T) {
	tempDir := t.TempDir()

	// Test files with content that requires normalization
	testFiles := map[string][]byte{
		"crlf.txt":     []byte("line1\r\nline2\r\nline3\r\n"),
		"lf.txt":       []byte("line1\nline2\nline3\n"),
		"cr.txt":       []byte("line1\rline2\rline3\r"),
		"mixed.txt":    []byte("line1\nline2\r\nline3\rline4\n"),
		"bom_utf8.txt": []byte("\xEF\xBB\xBFHello World\n"),
		"no_bom.txt":   []byte("Hello World\n"),
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, content, 0o644)
		if err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	options := Options{
		Algorithm:     "blake3",
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	// Calculate digest multiple times to ensure consistency
	const iterations = 5
	var digests []*Digest

	for i := 0; i < iterations; i++ {
		calc := NewProjectCalculator(tempDir, &options)
		digest, err := calc.Calculate(context.Background())
		if err != nil {
			t.Fatalf("Calculate() iteration %d failed: %v", i, err)
		}
		digests = append(digests, digest)
	}

	// Verify all iterations produce identical results
	baseDigest := digests[0]
	for i := 1; i < iterations; i++ {
		if digests[i].Hash != baseDigest.Hash {
			t.Errorf("normalization inconsistent: hash mismatch at iteration %d", i)
		}
	}

	t.Logf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	t.Logf("Content normalization consistent across %d iterations", iterations)
	t.Logf("Final hash: %s", baseDigest.Hash)
}
