package cache

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"testing"
	"time"
)

// helper to create a command that prints output and optional failure
func mockCmd(output string, fail bool) *exec.Cmd {
	script := fmt.Sprintf("printf %q", output)
	if fail {
		script += "; exit 1"
	}
	return exec.Command("sh", "-c", script)
}

func TestCapsuleCache_Exists(t *testing.T) {
	originalExec := execCommand
	defer func() { execCommand = originalExec }()

	tests := []struct {
		name       string
		mockOutput string
		want       bool
	}{
		{"image exists", "sha256:def456", true},
		{"image not exists", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCommand = func(name string, args ...string) *exec.Cmd {
				return mockCmd(tt.mockOutput, false)
			}
			cache := NewCapsuleCache("docker", "mitl-capsule:abc123")
			got, err := cache.Exists()
			if err != nil {
				t.Fatalf("Exists() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapsuleCache_MemoryCacheTTL(t *testing.T) {
	originalExec := execCommand
	defer func() { execCommand = originalExec }()
	originalNow := timeNow
	defer func() { timeNow = originalNow }()

	calls := 0
	execCommand = func(name string, args ...string) *exec.Cmd {
		calls++
		return mockCmd("sha", false)
	}
	base := time.Now()
	timeNow = func() time.Time { return base }

	cache := NewCapsuleCache("docker", "mitl-capsule:abc123")
	if _, err := cache.Exists(); err != nil {
		t.Fatalf("first Exists(): %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
	if _, err := cache.Exists(); err != nil {
		t.Fatalf("second Exists(): %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected cache hit, got %d calls", calls)
	}
	timeNow = func() time.Time { return base.Add(6 * time.Minute) }
	if _, err := cache.Exists(); err != nil {
		t.Fatalf("third Exists(): %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected cache expiration, got %d calls", calls)
	}
}

func TestCapsuleCache_ExistsWithDetailsAndDigest(t *testing.T) {
	originalExec := execCommand
	defer func() { execCommand = originalExec }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "images" {
			return mockCmd("sha256:abc", false)
		}
		img := ImageDetails{Created: "now", Size: 1, Architecture: "amd64", RepoDigests: []string{"repo@sha256:abc"}}
		b, _ := json.Marshal(img)
		return mockCmd(string(b), false)
	}
	cache := NewCapsuleCache("docker", "mitl-capsule:abc")
	exists, details, err := cache.ExistsWithDetails()
	if err != nil {
		t.Fatalf("ExistsWithDetails error: %v", err)
	}
	if !exists {
		t.Fatalf("expected exists true")
	}
	if details.Architecture != "amd64" {
		t.Fatalf("unexpected details: %+v", details)
	}
	if !cache.validateDigest("abc") {
		t.Fatalf("validateDigest returned false")
	}
}

func BenchmarkCapsuleCache_Exists(b *testing.B) {
	originalExec := execCommand
	defer func() { execCommand = originalExec }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return mockCmd("sha", false)
	}
	cache := NewCapsuleCache("docker", "mitl-capsule:abc")
	for i := 0; i < b.N; i++ {
		cache.InvalidateCache()
		if _, err := cache.Exists(); err != nil {
			b.Fatalf("Exists() error: %v", err)
		}
	}
}

func TestCapsuleCache_InvalidateCacheForcesRecheck(t *testing.T) {
	originalExec := execCommand
	defer func() { execCommand = originalExec }()
	originalNow := timeNow
	defer func() { timeNow = originalNow }()

	calls := 0
	execCommand = func(name string, args ...string) *exec.Cmd {
		calls++
		if calls == 1 {
			return mockCmd("sha1", false)
		}
		return mockCmd("sha2", false)
	}
	fixed := time.Now()
	timeNow = func() time.Time { return fixed }
	c := NewCapsuleCache("docker", "mitl-capsule:abc123")
	if ok, err := c.Exists(); err != nil || !ok {
		t.Fatalf("first Exists failed: ok=%v err=%v", ok, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 runtime call, got %d", calls)
	}
	c.InvalidateCache()
	if ok, err := c.Exists(); err != nil || !ok {
		t.Fatalf("second Exists failed after invalidate: ok=%v err=%v", ok, err)
	}
	if calls != 2 {
		t.Fatalf("expected second runtime call after invalidate, got %d", calls)
	}
}

func TestCapsuleCache_Exists_ErrorPath(t *testing.T) {
	originalExec := execCommand
	defer func() { execCommand = originalExec }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return mockCmd("", true)
	}
	c := NewCapsuleCache("docker", "mitl-capsule:err")
	ok, err := c.Exists()
	if err == nil || ok {
		t.Fatalf("expected error and ok=false, got ok=%v err=%v", ok, err)
	}
}

func TestCapsuleCache_ExistsWithDetails_InspectError(t *testing.T) {
	originalExec := execCommand
	defer func() { execCommand = originalExec }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "images" {
			return mockCmd("sha256:haveit", false)
		}
		// inspect fails
		return mockCmd("", true)
	}
	c := NewCapsuleCache("docker", "mitl-capsule:abc")
	ok, _, err := c.ExistsWithDetails()
	if err == nil || ok {
		t.Fatalf("expected inspect error and ok=false, got ok=%v err=%v", ok, err)
	}
}

func TestCapsuleCache_ExistsWithDetails_ParseError(t *testing.T) {
	originalExec := execCommand
	defer func() { execCommand = originalExec }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "images" {
			return mockCmd("sha256:haveit", false)
		}
		// invalid JSON from inspect
		return mockCmd("{not json}", false)
	}
	c := NewCapsuleCache("docker", "mitl-capsule:abc")
	ok, _, err := c.ExistsWithDetails()
	if err == nil {
		t.Fatalf("expected parse error, got nil")
	}
	if !ok {
		t.Fatalf("expected ok=true despite parse error")
	}
}

func TestValidateDigest_False_NoMatch(t *testing.T) {
	originalExec := execCommand
	defer func() { execCommand = originalExec }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "images" {
			return mockCmd("sha256:haveit", false)
		}
		// RepoDigests without expected digest
		img := ImageDetails{Created: "now", Size: 1, Architecture: "amd64", RepoDigests: []string{"repo@sha256:zzz"}}
		b, _ := json.Marshal(img)
		return mockCmd(string(b), false)
	}
	c := NewCapsuleCache("docker", "mitl-capsule:abc")
	if c.validateDigest("abc") {
		t.Fatalf("expected validateDigest to be false when digest does not match")
	}
}

func TestCapsuleCache_ExistsWithDetails_NotExists(t *testing.T) {
	originalExec := execCommand
	defer func() { execCommand = originalExec }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "images" {
			return mockCmd("", false)
		}
		return mockCmd("", false)
	}
	c := NewCapsuleCache("docker", "mitl-capsule:none")
	ok, _, err := c.ExistsWithDetails()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false when image not found")
	}
}
