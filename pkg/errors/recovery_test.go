package errors

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecover_NetworkRetry(t *testing.T) {
	r := NewRecoverer(true)
	err := New(ErrNetworkTimeout, "timeout")
	if rec := r.Recover(err); rec != nil {
		t.Fatalf("expected recovery to succeed for network timeout, got %v", rec)
	}
}

func TestRuntimeStartStrategy_FailsGracefully(t *testing.T) {
	r := NewRecoverer(true)
	err := New(ErrRuntimeNotRunning, "runtime down").WithContext("runtime", "podman")
	// We expect attempt to fail if podman isn't present; Recover should return non-nil (original error)
	if rec := r.Recover(err); rec == nil {
		t.Fatalf("expected recovery error when runtime start fails")
	}
}

func TestCacheClearStrategy_Attempt(t *testing.T) {
	// Point HOME to a temp dir to operate safely
	tmp := t.TempDir()
	old := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", old)

	// Create a bogus file in cache
	cacheDir := filepath.Join(tmp, ".mitl", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &CacheClearStrategy{}
	mitlErr := New(ErrCacheCorrupted, "cache corrupted")
	if err := s.Attempt(mitlErr); err != nil {
		t.Fatalf("cache clear attempt failed: %v", err)
	}
	// Ensure directory recreated
	if _, err := os.Stat(cacheDir); err != nil {
		t.Fatalf("expected cache directory to exist: %v", err)
	}
}

func TestDiskSpaceStrategy_AttemptEcho(t *testing.T) {
	s := &DiskSpaceStrategy{}
	err := s.Attempt(New(ErrDiskFull, "disk full").WithContext("runtime", "/bin/echo"))
	if err != nil {
		t.Fatalf("disk space strategy with echo should succeed: %v", err)
	}
}
