package commands

import (
	"bytes"
	"io"
	"os"
	"testing"

	"mitl/internal/digest"
)

func captureOut(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var b bytes.Buffer
	io.Copy(&b, r)
	return b.String()
}

func TestDigest_displayResultsAndFiles(t *testing.T) {
	dcmd := &DigestCommand{}
	d := &digest.Digest{
		Hash: "1234567890abcdef", Algorithm: "sha256", FileCount: 2, TotalSize: 10,
		Files: []digest.FileDigest{{Path: "a", Hash: "111111111111", Size: 5}, {Path: "b", Hash: "222222222222", Size: 5}},
	}
	out := captureOut(t, func() { cfg := digestConfig{verbose: true, showFiles: true}; dcmd.displayResults(d, &cfg) })
	if out == "" || !bytes.Contains([]byte(out), []byte("Digest")) {
		t.Fatalf("expected output to contain digest info, got: %s", out)
	}
}

func TestDigest_runLockfilesMode(t *testing.T) {
	dir := t.TempDir()
	// Create a simple lockfile
	os.WriteFile(dir+"/go.sum", []byte("a v1 h1:abc"), 0o644)
	cfg := digestConfig{rootDir: dir, verbose: true}
	dcmd := &DigestCommand{}
	out := captureOut(t, func() {
		_ = dcmd.runLockfilesMode(&cfg)
	})
	if out == "" {
		t.Fatalf("expected some output from lockfiles mode")
	}
}
