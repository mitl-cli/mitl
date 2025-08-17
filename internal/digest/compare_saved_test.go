package digest

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoadDigest(t *testing.T) {
	d := &Digest{Hash: "abcd", Algorithm: "sha256", Files: []FileDigest{{Path: "a", Hash: "1", Size: 1}}}
	dir := t.TempDir()
	p := filepath.Join(dir, "d.json")
	if err := SaveDigest(d, p); err != nil {
		t.Fatalf("save: %v", err)
	}
	ld, err := LoadDigest(p)
	if err != nil || ld.Hash != d.Hash {
		t.Fatalf("load mismatch: %v %+v", err, ld)
	}
	// CompareWithSaved convenience path
	comp, err := CompareWithSaved(p, d)
	if err != nil || !comp.Identical {
		t.Fatalf("compare saved: %v %+v", err, comp)
	}
	// Nil save error path tested elsewhere; ensure cannot write into dir
	if err := SaveDigest(d, dir); err == nil {
		t.Fatalf("expected error for invalid path")
	}
}
