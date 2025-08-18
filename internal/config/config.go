// Package config provides configuration management for the mitl tool.
// It handles loading, saving, and managing user preferences including
// container runtime selections and build performance tracking.
//
// Configuration is stored in JSON format at ~/.mitl.json and includes:
//   - Preferred build CLI (docker, podman, container, etc.)
//   - Preferred run CLI (may differ from build CLI)
//   - Build performance metrics for optimization
//
// The package gracefully handles missing configuration files by
// returning empty configurations, allowing the tool to work with
// sensible defaults when no explicit configuration exists.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config holds user preferences for container runtimes and misc state.
type Config struct {
	BuildCLI         string             `json:"build_cli"`
	RunCLI           string             `json:"run_cli"`
	LastBuildSeconds map[string]float64 `json:"last_build_seconds,omitempty"`
}

// Path returns the absolute path to the mitl configuration file (~/.mitl.json).
func Path() string {
	home := os.Getenv("HOME")
	if home == "" {
		if wd, _ := os.Getwd(); wd != "" {
			return filepath.Join(wd, ".mitl.json")
		}
	}
	return filepath.Join(home, ".mitl.json")
}

// Load reads configuration from disk. If missing, returns an empty config and nil error.
func Load() (*Config, error) {
	var cfg Config
	p := Path()
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return &Config{}, nil // treat parse issues as empty config (non-fatal)
	}
	return &cfg, nil
}

// Save writes configuration to disk.
func Save(cfg *Config) error {
	p := Path()
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}
