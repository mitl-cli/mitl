package build

import (
	"mitl/internal/detector"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateDockerfile_Laravel(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"require":{"laravel/framework":"^10.0"}}`), 0644)
	os.WriteFile(filepath.Join(dir, "artisan"), []byte(""), 0644)

	det := detector.NewProjectDetector(dir)
	det.Detect()
	gen := NewDockerfileGenerator(det)
	df, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(df, "FROM php:") {
		t.Fatalf("expected PHP base image in dockerfile, got: %s", df)
	}
}

func TestGenerateDockerfile_Node(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"next":"13.0.0"}}`), 0644)
	os.WriteFile(filepath.Join(dir, "next.config.js"), []byte(""), 0644)

	det := detector.NewProjectDetector(dir)
	det.Detect()
	gen := NewDockerfileGenerator(det)
	df, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(df, "FROM node:") {
		t.Fatalf("expected Node base image in dockerfile, got: %s", df)
	}
}
