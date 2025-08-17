package commands

import (
	"strings"
	"testing"
)

func TestDigest_parseFlags_Basic(t *testing.T) {
	cmd := &DigestCommand{}
	cfg := cmd.parseFlags([]string{"--algorithm", "sha256", "--include-hidden", "--only-ext", ".go,.md", "--exclude-ext", "log", "--root", "."})
	if cfg.options.Algorithm != "sha256" || !cfg.options.IncludeHidden {
		t.Fatalf("unexpected options: %+v", cfg.options)
	}
	if len(cfg.options.IncludePattern) == 0 || len(cfg.options.ExcludePattern) == 0 {
		t.Fatalf("expected include/exclude patterns to be set: %+v", cfg.options)
	}
	// Ensure patterns were converted with wildcards
	joined := strings.Join(cfg.options.IncludePattern, ",")
	if !strings.Contains(joined, "*.go") || !strings.Contains(joined, "*.md") {
		t.Errorf("expected wildcard extensions, got %v", cfg.options.IncludePattern)
	}
}
