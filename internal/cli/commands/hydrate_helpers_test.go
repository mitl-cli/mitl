package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHydrate_ConfigHelpers(t *testing.T) {
	tmp := t.TempDir()
	old := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", old)
	// Save and load config
	cfg := Config{BuildCLI: "/bin/echo", RunCLI: "/bin/echo", LastBuildSeconds: map[string]float64{"k": 1.23}}
	saveConfig(cfg)
	got := loadConfig()
	if got.BuildCLI == "" || got.RunCLI == "" {
		t.Fatalf("expected config to load")
	}
	// Verify path
	if p := configPath(); filepath.Dir(p) != tmp {
		t.Fatalf("expected config in HOME: %s", p)
	}
}

func TestFindBuildAndRunCLI_Env(t *testing.T) {
	os.Setenv("MITL_BUILD_CLI", "/bin/echo")
	if v := findBuildCLI(); v != "/bin/echo" {
		t.Fatalf("findBuildCLI env override failed: %s", v)
	}
	os.Setenv("MITL_RUN_CLI", "/bin/echo")
	if v := findRunCLI(); v != "/bin/echo" {
		t.Fatalf("findRunCLI env override failed: %s", v)
	}
}
