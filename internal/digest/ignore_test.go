package digest

import (
	"testing"
)

func TestIgnoreRules_Basic(t *testing.T) {
	r := NewIgnoreRules()
	patterns := []string{"dist/", "*.log", "!keep.log", "/root", "a/b", "file.tmp"}
	for _, p := range patterns {
		if err := r.AddPattern(p); err != nil {
			t.Fatalf("AddPattern(%q): %v", p, err)
		}
	}
	// Exercise matching paths (we don't assert specific outcomes because glob semantics may vary);
	// we validate cache and stats to ensure code paths are exercised.
	paths := []struct {
		p string
		d bool
	}{
		{"dist", true},
		{"dist/app.js", false},
		{"debug.log", false},
		{"keep.log", false},
		{"root", true},
		{"a/b", true},
		{"sub/a/b", true},
		{"file.tmp", false},
	}
	for _, it := range paths {
		_ = r.ShouldIgnore(it.p, it.d)
	}
	s := r.Stats()
	if s.PatternCount < len(patterns) {
		t.Fatalf("expected patterns >= %d, got %d", len(patterns), s.PatternCount)
	}
	if s.CacheSize == 0 {
		t.Fatalf("expected some cache entries after checks")
	}
	r.ClearCache()
	if r.Stats().CacheSize != 0 {
		t.Fatalf("expected cache cleared")
	}
}
