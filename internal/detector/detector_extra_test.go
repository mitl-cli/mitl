package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetector_PythonAndNodeRefinements(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname='x'\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "manage.py"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "next.config.js"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"next":"13"}}`), 0o644)

	d := NewProjectDetector(dir)
	d.Detect()
	if d.Type != TypeNodeNext { // next config should refine
		t.Fatalf("expected node-next, got %s", d.Type)
	}
	if !d.ValidatePyProject(filepath.Join(dir, "pyproject.toml")) {
		t.Fatalf("pyproject should validate")
	}
}

func TestDetector_ScanPHPAndExtractors(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"require":{"php":"^8.2"}}`), 0o644)
	os.WriteFile(filepath.Join(dir, "test.php"), []byte("<?php curl_init(); json_encode([]); ?>"), 0o644)
	d := NewProjectDetector(dir)
	d.Type = TypePHPGeneric
	d.Detect()
	// Should pick up curl/json
	exts := d.Dependencies.PHP.Extensions
	if !ContainsString(exts, "curl") || !ContainsString(exts, "json") {
		t.Fatalf("expected curl/json extensions, got %v", exts)
	}
	if ExtractPHPVersion(">=8.3") == "" || ExtractNodeVersion(">=20.1") == "" {
		t.Fatalf("version extractors should return values")
	}
}

func TestDetector_CheckFlaskImports(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.py")
	os.WriteFile(path, []byte("from flask import Flask"), 0o644)
	d := NewProjectDetector(dir)
	if !d.CheckFlaskImports(path) {
		t.Fatalf("expected flask import detection")
	}
}
