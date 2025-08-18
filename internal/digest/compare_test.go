package digest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompare(t *testing.T) {
	// Create test digests
	baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	digest1 := &Digest{
		Hash:      "abc123",
		Algorithm: "blake3",
		Timestamp: baseTime,
		FileCount: 3,
		TotalSize: 1000,
		Files: []FileDigest{
			{Path: "file1.txt", Hash: "hash1", Size: 100},
			{Path: "file2.txt", Hash: "hash2", Size: 200},
			{Path: "file3.txt", Hash: "hash3", Size: 700},
		},
		Options: Options{Algorithm: "blake3"},
	}

	digest2 := &Digest{
		Hash:      "def456",
		Algorithm: "blake3",
		Timestamp: baseTime.Add(time.Hour),
		FileCount: 4,
		TotalSize: 1500,
		Files: []FileDigest{
			{Path: "file1.txt", Hash: "hash1", Size: 100},     // unchanged
			{Path: "file2.txt", Hash: "hash2_new", Size: 250}, // modified
			{Path: "file3.txt", Hash: "hash3", Size: 700},     // unchanged
			{Path: "file4.txt", Hash: "hash4", Size: 450},     // added
		},
		Options: Options{Algorithm: "blake3"},
	}

	tests := []struct {
		name     string
		digest1  *Digest
		digest2  *Digest
		validate func(t *testing.T, comp *Comparison)
	}{
		{
			name:    "identical digests",
			digest1: digest1,
			digest2: digest1, // same digest
			validate: func(t *testing.T, comp *Comparison) {
				if !comp.Identical {
					t.Error("expected identical digests")
				}
				if len(comp.Added) != 0 {
					t.Errorf("expected no added files, got %d", len(comp.Added))
				}
				if len(comp.Modified) != 0 {
					t.Errorf("expected no modified files, got %d", len(comp.Modified))
				}
				if len(comp.Removed) != 0 {
					t.Errorf("expected no removed files, got %d", len(comp.Removed))
				}
			},
		},
		{
			name:    "different digests with changes",
			digest1: digest1,
			digest2: digest2,
			validate: func(t *testing.T, comp *Comparison) {
				if comp.Identical {
					t.Error("expected non-identical digests")
				}
				if len(comp.Added) != 1 {
					t.Errorf("expected 1 added file, got %d", len(comp.Added))
				} else if comp.Added[0] != "file4.txt" {
					t.Errorf("expected file4.txt to be added, got %s", comp.Added[0])
				}
				if len(comp.Modified) != 1 {
					t.Errorf("expected 1 modified file, got %d", len(comp.Modified))
				} else if comp.Modified[0] != "file2.txt" {
					t.Errorf("expected file2.txt to be modified, got %s", comp.Modified[0])
				}
				if len(comp.Removed) != 0 {
					t.Errorf("expected no removed files, got %d", len(comp.Removed))
				}
			},
		},
		{
			name:    "files removed",
			digest1: digest2,
			digest2: digest1, // reverse comparison
			validate: func(t *testing.T, comp *Comparison) {
				if comp.Identical {
					t.Error("expected non-identical digests")
				}
				if len(comp.Removed) != 1 {
					t.Errorf("expected 1 removed file, got %d", len(comp.Removed))
				} else if comp.Removed[0] != "file4.txt" {
					t.Errorf("expected file4.txt to be removed, got %s", comp.Removed[0])
				}
				if len(comp.Modified) != 1 {
					t.Errorf("expected 1 modified file, got %d", len(comp.Modified))
				}
				if len(comp.Added) != 0 {
					t.Errorf("expected no added files, got %d", len(comp.Added))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := Compare(tt.digest1, tt.digest2)
			if comp == nil {
				t.Fatal("Compare() returned nil")
			}
			tt.validate(t, comp)
		})
	}
}

func TestComparison_Summary(t *testing.T) {
	tests := []struct {
		name       string
		comparison Comparison
		expected   string
	}{
		{
			name: "identical",
			comparison: Comparison{
				Identical: true,
			},
			expected: "Digests are identical",
		},
		{
			name: "only additions",
			comparison: Comparison{
				Identical: false,
				Added:     []string{"file1.txt", "file2.txt"},
			},
			expected: "2 files added",
		},
		{
			name: "only modifications",
			comparison: Comparison{
				Identical: false,
				Modified:  []string{"file1.txt"},
			},
			expected: "1 file modified",
		},
		{
			name: "only removals",
			comparison: Comparison{
				Identical: false,
				Removed:   []string{"file1.txt", "file2.txt", "file3.txt"},
			},
			expected: "3 files removed",
		},
		{
			name: "mixed changes",
			comparison: Comparison{
				Identical: false,
				Added:     []string{"new.txt"},
				Modified:  []string{"changed.txt", "updated.txt"},
				Removed:   []string{"deleted.txt"},
			},
			expected: "1 file added, 2 files modified, 1 file removed",
		},
		{
			name: "complex changes",
			comparison: Comparison{
				Identical: false,
				Added:     []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt"},
				Modified:  []string{"modified.txt"},
				Removed:   []string{"x.txt", "y.txt"},
			},
			expected: "5 files added, 1 file modified, 2 files removed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.comparison.Summary()
			if result != tt.expected {
				t.Errorf("Summary() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSaveDigest_LoadDigest(t *testing.T) {
	tempDir := t.TempDir()
	digestPath := filepath.Join(tempDir, "test.digest.json")

	// Create test digest
	original := &Digest{
		Hash:      "test-hash-123",
		Algorithm: "blake3",
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		FileCount: 2,
		TotalSize: 500,
		Files: []FileDigest{
			{Path: "main.go", Hash: "main-hash", Size: 300},
			{Path: "util.go", Hash: "util-hash", Size: 200},
		},
		Options: Options{
			Algorithm:     "blake3",
			MaxFileSize:   10485760,
			IncludeHidden: false,
			LockfilesOnly: false,
		},
	}

	// Test save
	err := SaveDigest(original, digestPath)
	if err != nil {
		t.Fatalf("SaveDigest() failed: %v", err)
	}

	// Verify file exists
	if _, statErr := os.Stat(digestPath); os.IsNotExist(statErr) {
		t.Fatal("digest file was not created")
	}

	// Test load
	loaded, err := LoadDigest(digestPath)
	if err != nil {
		t.Fatalf("LoadDigest() failed: %v", err)
	}

	// Compare loaded digest with original
	if loaded.Hash != original.Hash {
		t.Errorf("hash mismatch: got %s, want %s", loaded.Hash, original.Hash)
	}
	if loaded.Algorithm != original.Algorithm {
		t.Errorf("algorithm mismatch: got %s, want %s", loaded.Algorithm, original.Algorithm)
	}
	if loaded.FileCount != original.FileCount {
		t.Errorf("file count mismatch: got %d, want %d", loaded.FileCount, original.FileCount)
	}
	if loaded.TotalSize != original.TotalSize {
		t.Errorf("total size mismatch: got %d, want %d", loaded.TotalSize, original.TotalSize)
	}
	if len(loaded.Files) != len(original.Files) {
		t.Errorf("files length mismatch: got %d, want %d", len(loaded.Files), len(original.Files))
	}

	// Test timestamp preservation (should be close due to JSON precision)
	timeDiff := loaded.Timestamp.Sub(original.Timestamp)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > time.Second {
		t.Errorf("timestamp difference too large: %v", timeDiff)
	}
}

func TestSaveDigest_ErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		digest  *Digest
		path    string
		wantErr bool
	}{
		{
			name:    "nil digest",
			digest:  nil,
			path:    "test.json",
			wantErr: true,
		},
		{
			name: "valid digest",
			digest: &Digest{
				Hash:      "test",
				Algorithm: "blake3",
				Timestamp: time.Now(),
			},
			path:    filepath.Join(t.TempDir(), "valid.json"),
			wantErr: false,
		},
		{
			name: "invalid path",
			digest: &Digest{
				Hash:      "test",
				Algorithm: "blake3",
				Timestamp: time.Now(),
			},
			path:    "/invalid/path/that/does/not/exist/file.json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SaveDigest(tt.digest, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveDigest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadDigest_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		setup   func() string
		wantErr bool
	}{
		{
			name: "non-existent file",
			setup: func() string {
				return "/non/existent/file.json"
			},
			wantErr: true,
		},
		{
			name: "invalid JSON",
			setup: func() string {
				path := filepath.Join(tempDir, "invalid.json")
				os.WriteFile(path, []byte("invalid json content"), 0o644)
				return path
			},
			wantErr: true,
		},
		{
			name: "valid JSON file",
			setup: func() string {
				path := filepath.Join(tempDir, "valid.json")
				digest := &Digest{
					Hash:      "test",
					Algorithm: "blake3",
					Timestamp: time.Now(),
				}
				data, _ := json.Marshal(digest)
				os.WriteFile(path, data, 0o644)
				return path
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			_, err := LoadDigest(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadDigest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompareDirectories(t *testing.T) {
	// Create two temporary directories with different content
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Directory 1 files
	files1 := map[string]string{
		"main.go":   "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}\n",
		"go.mod":    "module test\n\ngo 1.21\n",
		"README.md": "# Test Project\n",
	}

	// Directory 2 files (with changes)
	files2 := map[string]string{
		"main.go":  "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n", // modified
		"go.mod":   "module test\n\ngo 1.21\n",                                         // unchanged
		"utils.go": "package main\n\nfunc Helper() {}\n",                               // added
		// README.md removed
	}

	// Create files in dir1
	for path, content := range files1 {
		fullPath := filepath.Join(dir1, path)
		os.MkdirAll(filepath.Dir(fullPath), 0o755)
		os.WriteFile(fullPath, []byte(content), 0o644)
	}

	// Create files in dir2
	for path, content := range files2 {
		fullPath := filepath.Join(dir2, path)
		os.MkdirAll(filepath.Dir(fullPath), 0o755)
		os.WriteFile(fullPath, []byte(content), 0o644)
	}

	options := Options{
		Algorithm:     "blake3",
		MaxFileSize:   10485760,
		IncludeHidden: false,
		LockfilesOnly: false,
	}

	// Calculate digests for both directories
	calc1 := NewProjectCalculator(dir1, &options)
	digest1, err := calc1.Calculate(context.Background())
	if err != nil {
		t.Fatalf("failed to calculate digest for dir1: %v", err)
	}

	calc2 := NewProjectCalculator(dir2, &options)
	digest2, err := calc2.Calculate(context.Background())
	if err != nil {
		t.Fatalf("failed to calculate digest for dir2: %v", err)
	}

	// Compare the digests
	comparison := Compare(digest1, digest2)

	// Verify the comparison results
	if comparison.Identical {
		t.Error("expected directories to be different")
	}

	// Check for added file
	if len(comparison.Added) != 1 || comparison.Added[0] != "utils.go" {
		t.Errorf("expected utils.go to be added, got: %v", comparison.Added)
	}

	// Check for modified file
	if len(comparison.Modified) != 1 || comparison.Modified[0] != "main.go" {
		t.Errorf("expected main.go to be modified, got: %v", comparison.Modified)
	}

	// Check for removed file
	if len(comparison.Removed) != 1 || comparison.Removed[0] != "README.md" {
		t.Errorf("expected README.md to be removed, got: %v", comparison.Removed)
	}

	// Test summary
	summary := comparison.Summary()
	expectedSummary := "1 file added, 1 file modified, 1 file removed"
	if summary != expectedSummary {
		t.Errorf("summary mismatch: got %q, want %q", summary, expectedSummary)
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()

	// Create a comprehensive test digest
	original := &Digest{
		Hash:      "comprehensive-test-hash-with-special-chars-äöü",
		Algorithm: "sha256",
		Timestamp: time.Now().UTC().Truncate(time.Millisecond), // Truncate for JSON precision
		FileCount: 100,
		TotalSize: 1024 * 1024 * 5, // 5MB
		Files: []FileDigest{
			{Path: "src/main.go", Hash: "main-file-hash", Size: 1024},
			{Path: "test/main_test.go", Hash: "test-file-hash", Size: 2048},
			{Path: "docs/README.md", Hash: "readme-hash", Size: 512},
			{Path: "config/app.yaml", Hash: "config-hash", Size: 256},
			{Path: "scripts/build.sh", Hash: "script-hash", Size: 128},
		},
		Options: Options{
			Algorithm:      "sha256",
			MaxFileSize:    1048576,
			IncludeHidden:  true,
			LockfilesOnly:  false,
			IncludePattern: []string{"*.go", "*.md", "*.yaml"},
			ExcludePattern: []string{"*.tmp", "*.log"},
		},
	}

	// Save and load multiple times to test consistency
	for i := 0; i < 3; i++ {
		digestPath := filepath.Join(tempDir, "round_trip_test.json")

		// Save
		err := SaveDigest(original, digestPath)
		if err != nil {
			t.Fatalf("SaveDigest() iteration %d failed: %v", i, err)
		}

		// Load
		loaded, err := LoadDigest(digestPath)
		if err != nil {
			t.Fatalf("LoadDigest() iteration %d failed: %v", i, err)
		}

		// Deep comparison
		if loaded.Hash != original.Hash {
			t.Errorf("iteration %d: hash mismatch", i)
		}
		if loaded.Algorithm != original.Algorithm {
			t.Errorf("iteration %d: algorithm mismatch", i)
		}
		if loaded.FileCount != original.FileCount {
			t.Errorf("iteration %d: file count mismatch", i)
		}
		if loaded.TotalSize != original.TotalSize {
			t.Errorf("iteration %d: total size mismatch", i)
		}

		// Verify options are preserved
		if loaded.Options.Algorithm != original.Options.Algorithm {
			t.Errorf("iteration %d: options algorithm mismatch", i)
		}
		if loaded.Options.MaxFileSize != original.Options.MaxFileSize {
			t.Errorf("iteration %d: options max file size mismatch", i)
		}
		if loaded.Options.IncludeHidden != original.Options.IncludeHidden {
			t.Errorf("iteration %d: options include hidden mismatch", i)
		}

		// Update original for next iteration to test that changes are preserved
		original = loaded
		original.Hash = original.Hash + "-updated"
	}
}

func BenchmarkCompare(b *testing.B) {
	// Create test digests with many files
	baseTime := time.Now()

	// Generate many files for realistic comparison
	files1 := make([]FileDigest, 1000)
	files2 := make([]FileDigest, 1000)

	for i := 0; i < 1000; i++ {
		files1[i] = FileDigest{
			Path: filepath.Join("src", "file_"+string(rune(i))+".go"),
			Hash: "hash" + string(rune(i)),
			Size: int64(100 + i),
		}
		// Make some changes for realistic comparison
		if i%10 == 0 {
			// Every 10th file is modified
			files2[i] = FileDigest{
				Path: files1[i].Path,
				Hash: "modified_" + files1[i].Hash,
				Size: files1[i].Size + 10,
			}
		} else {
			files2[i] = files1[i]
		}
	}

	digest1 := &Digest{
		Hash:      "digest1-hash",
		Algorithm: "blake3",
		Timestamp: baseTime,
		FileCount: 1000,
		TotalSize: 150500,
		Files:     files1,
		Options:   Options{Algorithm: "blake3"},
	}

	digest2 := &Digest{
		Hash:      "digest2-hash",
		Algorithm: "blake3",
		Timestamp: baseTime.Add(time.Hour),
		FileCount: 1000,
		TotalSize: 151500,
		Files:     files2,
		Options:   Options{Algorithm: "blake3"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		comparison := Compare(digest1, digest2)
		if comparison == nil {
			b.Fatal("Compare returned nil")
		}
	}
}

func BenchmarkSaveDigest(b *testing.B) {
	tempDir := b.TempDir()

	// Create a large digest for benchmarking
	files := make([]FileDigest, 1000)
	for i := 0; i < 1000; i++ {
		files[i] = FileDigest{
			Path: filepath.Join("src", "file_"+string(rune(i))+".go"),
			Hash: "hash-content-for-file-" + string(rune(i)),
			Size: int64(100 + i),
		}
	}

	digest := &Digest{
		Hash:      "large-digest-hash-for-benchmarking-performance",
		Algorithm: "blake3",
		Timestamp: time.Now(),
		FileCount: 1000,
		TotalSize: 150500,
		Files:     files,
		Options: Options{
			Algorithm:     "blake3",
			MaxFileSize:   10485760,
			IncludeHidden: false,
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := filepath.Join(tempDir, "bench_digest.json")
		err := SaveDigest(digest, path)
		if err != nil {
			b.Fatalf("SaveDigest failed: %v", err)
		}
		os.Remove(path) // Clean up for next iteration
	}
}

func BenchmarkLoadDigest(b *testing.B) {
	tempDir := b.TempDir()
	path := filepath.Join(tempDir, "load_bench.json")

	// Create a large digest file for benchmarking
	files := make([]FileDigest, 1000)
	for i := 0; i < 1000; i++ {
		files[i] = FileDigest{
			Path: filepath.Join("src", "file_"+string(rune(i))+".go"),
			Hash: "hash-content-for-file-" + string(rune(i)),
			Size: int64(100 + i),
		}
	}

	digest := &Digest{
		Hash:      "large-digest-hash-for-benchmarking-performance",
		Algorithm: "blake3",
		Timestamp: time.Now(),
		FileCount: 1000,
		TotalSize: 150500,
		Files:     files,
		Options: Options{
			Algorithm:     "blake3",
			MaxFileSize:   10485760,
			IncludeHidden: false,
		},
	}

	// Save the digest once for loading benchmark
	err := SaveDigest(digest, path)
	if err != nil {
		b.Fatalf("Failed to save digest for benchmark: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := LoadDigest(path)
		if err != nil {
			b.Fatalf("LoadDigest failed: %v", err)
		}
	}
}
