package digest

import "testing"

func TestComparison_SummaryTwoParts(t *testing.T) {
	d1 := &Digest{Hash: "a", Algorithm: "sha256", Files: []FileDigest{{Path: "a", Hash: "1", Size: 1}}}
	d2 := &Digest{Hash: "b", Algorithm: "sha256", Files: []FileDigest{{Path: "a", Hash: "2", Size: 1}}}
	c := Compare(d1, d2)
	s := c.Summary()
	if s == "" {
		t.Fatalf("expected non-empty summary")
	}
}
