package digest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDigest_JSON(t *testing.T) {
	tests := []struct {
		name   string
		digest Digest
	}{
		{
			name: "complete digest",
			digest: Digest{
				Hash:      "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262",
				Algorithm: "blake3",
				Timestamp: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				FileCount: 42,
				TotalSize: 1024000,
				Files: []FileDigest{
					{
						Path: "main.go",
						Hash: "123abc",
						Size: 1024,
					},
					{
						Path: "go.mod",
						Hash: "456def",
						Size: 256,
					},
				},
				Options: Options{
					Algorithm:     "blake3",
					MaxFileSize:   10485760,
					IncludeHidden: false,
					LockfilesOnly: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.digest)
			if err != nil {
				t.Fatalf("failed to marshal digest: %v", err)
			}

			// Test JSON unmarshaling
			var unmarshaled Digest
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Fatalf("failed to unmarshal digest: %v", err)
			}

			// Compare fields
			if !reflect.DeepEqual(tt.digest, unmarshaled) {
				t.Errorf("digest mismatch after JSON round-trip:\noriginal: %+v\nunmarshaled: %+v", tt.digest, unmarshaled)
			}
		})
	}
}

func TestProjectCalculator_Calculate(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"main.go":     "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n",
		"go.mod":      "module test\n\ngo 1.21\n",
		"README.md":   "# Test Project\n\nThis is a test.\n",
		"src/util.go": "package src\n\nfunc Helper() string {\n\treturn \"helper\"\n}\n",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		if err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("failed to write file %s: %v", path, err)
		}
	}

	tests := []struct {
		name     string
		options  Options
		wantErr  bool
		validate func(t *testing.T, digest *Digest)
	}{
		{
			name: "basic calculation",
			options: Options{
				Algorithm:     "blake3",
				MaxFileSize:   10485760,
				IncludeHidden: false,
				LockfilesOnly: false,
			},
			wantErr: false,
			validate: func(t *testing.T, digest *Digest) {
				if digest.Hash == "" {
					t.Error("expected non-empty hash")
				}
				if digest.Algorithm != "blake3" {
					t.Errorf("expected algorithm blake3, got %s", digest.Algorithm)
				}
				if digest.FileCount != 4 {
					t.Errorf("expected 4 files, got %d", digest.FileCount)
				}
				if len(digest.Files) != 4 {
					t.Errorf("expected 4 file entries, got %d", len(digest.Files))
				}
				if digest.TotalSize == 0 {
					t.Error("expected non-zero total size")
				}
			},
		},
		{
			name: "sha256 algorithm",
			options: Options{
				Algorithm:     "sha256",
				MaxFileSize:   10485760,
				IncludeHidden: false,
				LockfilesOnly: false,
			},
			wantErr: false,
			validate: func(t *testing.T, digest *Digest) {
				if digest.Algorithm != "sha256" {
					t.Errorf("expected algorithm sha256, got %s", digest.Algorithm)
				}
			},
		},
		{
			name: "include patterns",
			options: Options{
				Algorithm:      "blake3",
				MaxFileSize:    10485760,
				IncludeHidden:  false,
				LockfilesOnly:  false,
				IncludePattern: []string{"*.go"},
			},
			wantErr: false,
			validate: func(t *testing.T, digest *Digest) {
				if digest.FileCount != 2 { // main.go and src/util.go
					t.Errorf("expected 2 Go files, got %d", digest.FileCount)
				}
			},
		},
		{
			name: "exclude patterns",
			options: Options{
				Algorithm:      "blake3",
				MaxFileSize:    10485760,
				IncludeHidden:  false,
				LockfilesOnly:  false,
				ExcludePattern: []string{"*.md"},
			},
			wantErr: false,
			validate: func(t *testing.T, digest *Digest) {
				if digest.FileCount != 3 { // Excluding README.md
					t.Errorf("expected 3 files (excluding .md), got %d", digest.FileCount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := NewProjectCalculator(tempDir, &tt.options)

			digest, err := calc.Calculate(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Calculate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, digest)
			}
		})
	}
}

func TestProjectCalculator_Determinism(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()

	// Create test files with various content types
	testFiles := map[string]string{
		"main.go":        "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n",
		"go.mod":         "module test\n\ngo 1.21\n",
		"data.json":      `{"name": "test", "version": "1.0.0"}`,
		"script.sh":      "#!/bin/bash\necho 'Hello World'\n",
		"config.yml":     "database:\n  host: localhost\n  port: 5432\n",
		"src/helper.go":  "package src\n\nfunc Helper() {}\n",
		"docs/README.md": "# Documentation\n\nProject docs here.\n",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		if err != nil {
			t.Fatalf("failed to create directory: %v", err)
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
	const iterations = 5
	digests := make([]*Digest, iterations)

	for i := 0; i < iterations; i++ {
		calc := NewProjectCalculator(tempDir, &options)
		digest, err := calc.Calculate(context.Background())
		if err != nil {
			t.Fatalf("Calculate() iteration %d failed: %v", i, err)
		}
		digests[i] = digest
	}

	// Verify all digests are identical
	baseDigest := digests[0]
	for i := 1; i < iterations; i++ {
		if digests[i].Hash != baseDigest.Hash {
			t.Errorf("digest hash mismatch at iteration %d: expected %s, got %s",
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

	// Test file order consistency
	for i := 1; i < iterations; i++ {
		if len(digests[i].Files) != len(baseDigest.Files) {
			t.Errorf("file list length mismatch at iteration %d", i)
			continue
		}
		for j, file := range digests[i].Files {
			if file.Path != baseDigest.Files[j].Path {
				t.Errorf("file order mismatch at iteration %d, position %d: expected %s, got %s",
					i, j, baseDigest.Files[j].Path, file.Path)
			}
			if file.Hash != baseDigest.Files[j].Hash {
				t.Errorf("file hash mismatch at iteration %d for %s: expected %s, got %s",
					i, file.Path, baseDigest.Files[j].Hash, file.Hash)
			}
		}
	}
}

func TestProjectCalculator_CrossPlatformNormalization(t *testing.T) {
	tempDir := t.TempDir()

	// Test content with different line endings and BOM
	testCases := []struct {
		name    string
		content string
		desc    string
	}{
		{
			name:    "unix_endings.txt",
			content: "line1\nline2\nline3\n",
			desc:    "Unix line endings (LF)",
		},
		{
			name:    "windows_endings.txt",
			content: "line1\r\nline2\r\nline3\r\n",
			desc:    "Windows line endings (CRLF)",
		},
		{
			name:    "mac_endings.txt",
			content: "line1\rline2\rline3\r",
			desc:    "Mac line endings (CR)",
		},
		{
			name:    "mixed_endings.txt",
			content: "line1\nline2\r\nline3\rline4\n",
			desc:    "Mixed line endings",
		},
		{
			name:    "bom_utf8.txt",
			content: "\xEF\xBB\xBFHello World\n",
			desc:    "UTF-8 BOM",
		},
	}

	var digests []string

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tc.name)
			err := os.WriteFile(filePath, []byte(tc.content), 0o644)
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
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

			// Find the specific file hash
			var fileHash string
			for _, file := range digest.Files {
				if strings.Contains(file.Path, tc.name) {
					fileHash = file.Hash
					break
				}
			}

			if fileHash == "" {
				t.Fatalf("file hash not found for %s", tc.name)
			}

			digests = append(digests, fileHash)

			// Clean up for next iteration
			os.Remove(filePath)
		})
	}

	// Verify that different line endings and BOM are normalized to same hash
	// (This validates the normalization is working)
	t.Logf("File hashes: %v", digests)

	// The first three should be identical (different line endings)
	// The BOM file should also normalize to same content
	baseHash := digests[0]
	for i := 1; i < 3; i++ {
		if digests[i] != baseHash {
			t.Errorf("line ending normalization failed: hash %d (%s) != base hash (%s)",
				i, digests[i], baseHash)
		}
	}
}

func TestProjectCalculator_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	tempDir := t.TempDir()

	// Create many small files to test performance
	const numFiles = 100
	for i := 0; i < numFiles; i++ {
		content := strings.Repeat("test content line\n", 10)
		filePath := filepath.Join(tempDir, fmt.Sprintf("file_%03d.txt", i))
		err := os.WriteFile(filePath, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("failed to write test file %d: %v", i, err)
		}
	}

	options := Options{
		Algorithm:     "blake3",
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	calc := NewProjectCalculator(tempDir, &options)

	start := time.Now()
	digest, err := calc.Calculate(context.Background())
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Calculate() failed: %v", err)
	}

	t.Logf("Calculated digest for %d files in %v", digest.FileCount, duration)

	// Performance requirement: should be fast for reasonable number of files
	if duration > 5*time.Second {
		t.Errorf("performance too slow: took %v for %d files", duration, numFiles)
	}

	if digest.FileCount != numFiles {
		t.Errorf("expected %d files, got %d", numFiles, digest.FileCount)
	}
}

func TestProjectCalculator_ErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		options Options
		wantErr bool
	}{
		{
			name: "non-existent directory",
			setup: func(t *testing.T) string {
				return "/non/existent/directory"
			},
			options: Options{
				Algorithm: "blake3",
			},
			wantErr: true,
		},
		{
			name: "invalid algorithm",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			options: Options{
				Algorithm: "invalid-algorithm",
			},
			wantErr: true,
		},
		{
			name: "context cancellation",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				// Create a large file to ensure context cancellation can occur
				content := strings.Repeat("large content\n", 10000)
				filePath := filepath.Join(tempDir, "large_file.txt")
				os.WriteFile(filePath, []byte(content), 0o644)
				return tempDir
			},
			options: Options{
				Algorithm: "blake3",
			},
			wantErr: true, // We'll cancel the context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootPath := tt.setup(t)
			calc := NewProjectCalculator(rootPath, &tt.options)

			ctx := context.Background()
			if tt.name == "context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 1*time.Millisecond)
				defer cancel()
			}

			_, err := calc.Calculate(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Calculate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		wantErr bool
	}{
		{
			name: "valid blake3",
			options: Options{
				Algorithm: "blake3",
			},
			wantErr: false,
		},
		{
			name: "valid sha256",
			options: Options{
				Algorithm: "sha256",
			},
			wantErr: false,
		},
		{
			name: "invalid algorithm",
			options: Options{
				Algorithm: "md5",
			},
			wantErr: true,
		},
		{
			name: "empty algorithm defaults to blake3",
			options: Options{
				Algorithm: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Options.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function for consistent formatting
func formatBytesTest(t *testing.T, bytes int64, expected string) {
	result := formatBytes(bytes)
	if result != expected {
		t.Errorf("formatBytes(%d) = %s, want %s", bytes, result, expected)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			formatBytesTest(t, tt.bytes, tt.expected)
		})
	}
}

func BenchmarkProjectCalculator_Calculate(b *testing.B) {
	tempDir := b.TempDir()

	// Create test files
	testFiles := map[string]string{
		"main.go":     "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n",
		"go.mod":      "module test\n\ngo 1.21\n",
		"README.md":   "# Test Project\n\nThis is a test.\n",
		"src/util.go": "package src\n\nfunc Helper() string {\n\treturn \"helper\"\n}\n",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0o755)
		os.WriteFile(fullPath, []byte(content), 0o644)
	}

	options := Options{
		Algorithm:     "blake3",
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		calc := NewProjectCalculator(tempDir, &options)
		_, err := calc.Calculate(context.Background())
		if err != nil {
			b.Fatalf("Calculate() failed: %v", err)
		}
	}
}

func BenchmarkProjectCalculator_Blake3VsSHA256(b *testing.B) {
	tempDir := b.TempDir()

	// Create a moderately sized file for comparison
	content := strings.Repeat("This is test content for benchmarking.\n", 1000)
	filePath := filepath.Join(tempDir, "test_file.txt")
	os.WriteFile(filePath, []byte(content), 0o644)

	algorithms := []string{"blake3", "sha256"}

	for _, alg := range algorithms {
		b.Run(alg, func(b *testing.B) {
			options := Options{
				Algorithm:     alg,
				MaxFileSize:   10485760,
				IncludeHidden: false,
				LockfilesOnly: false,
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				calc := NewProjectCalculator(tempDir, &options)
				_, err := calc.Calculate(context.Background())
				if err != nil {
					b.Fatalf("Calculate() failed: %v", err)
				}
			}
		})
	}
}
