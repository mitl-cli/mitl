package digest

import "testing"

func TestCompare_AlgorithmOnlyChange(t *testing.T) {
	d1 := &Digest{Hash: "x", Algorithm: "sha256", Files: []FileDigest{{Path: "a", Hash: "1", Size: 1}}}
	d2 := &Digest{Hash: "y", Algorithm: "blake3", Files: []FileDigest{{Path: "a", Hash: "1", Size: 1}}}
	c := Compare(d1, d2)
	if c.Identical {
		t.Fatalf("expected not identical")
	}
	if len(c.Added)+len(c.Removed)+len(c.Modified) != 0 {
		t.Fatalf("expected no file diffs when only algorithm changes: %+v", c)
	}
}
