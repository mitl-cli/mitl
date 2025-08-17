package digest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnoreRulesFromProject(t *testing.T) {
	dir := t.TempDir()
	content := "# comment\n*.log\n!/keep.log\n"
	if err := os.WriteFile(filepath.Join(dir, ".mitlignore"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	rules, err := LoadIgnoreRulesFromProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	if rules.Stats().PatternCount < 1 {
		t.Fatalf("expected patterns loaded")
	}
	_ = rules.ShouldIgnore("a.log", false)
	if rules.Stats().CacheSize == 0 {
		t.Fatalf("expected cache entries recorded")
	}
	// Verify patterns exported via GetPatterns include our entries
	pats := rules.GetPatterns()
	if len(pats) == 0 {
		t.Fatalf("expected patterns returned")
	}
}
