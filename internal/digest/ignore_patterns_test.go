package digest

import "testing"

func TestIgnoreRules_PatternVariants(t *testing.T) {
	r := NewIgnoreRules()
	// Absolute-like pattern and no-slash pattern
	for _, p := range []string{"/build", "*.tmp", "logs/", "!keep.tmp"} {
		if err := r.AddPattern(p); err != nil {
			t.Fatalf("AddPattern %q: %v", p, err)
		}
	}
	// Exercise matches for files/dirs
	_ = r.ShouldIgnore("build", true)
	_ = r.ShouldIgnore("sub/build", true)
	_ = r.ShouldIgnore("a.tmp", false)
	_ = r.ShouldIgnore("logs", true)
	_ = r.ShouldIgnore("keep.tmp", false)
	if r.Stats().PatternCount < 4 {
		t.Fatalf("expected 4 patterns")
	}
}
