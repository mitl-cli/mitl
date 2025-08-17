// Package cache provides caching operations for mitl capsules.
package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// testable exec command wrapper
var execCommand = exec.Command

// testable time wrapper
var timeNow = time.Now

const cacheTTL = 5 * time.Minute

// CapsuleCache manages detection of existing capsules.
// It's thread-safe and maintains an in-memory cache of recent checks.
type CapsuleCache struct {
	runtime  string
	tag      string
	mu       sync.RWMutex
	memCache map[string]cacheEntry
}

type cacheEntry struct {
	exists    bool
	timestamp time.Time
}

// ImageDetails contains basic image metadata
type ImageDetails struct {
	Created      string   `json:"Created"`
	Size         int64    `json:"Size"`
	Architecture string   `json:"Architecture"`
	RepoDigests  []string `json:"RepoDigests"`
}

// Statistics contains cache statistics
type Statistics struct {
	Hits      int64
	Misses    int64
	Size      int64
	ItemCount int
}

// NewCapsuleCache creates a cache manager instance
func NewCapsuleCache(runtime, tag string) *CapsuleCache {
	return &CapsuleCache{
		runtime:  runtime,
		tag:      tag,
		memCache: make(map[string]cacheEntry),
	}
}

// Exists checks if the capsule exists using the configured runtime.
// First checks in-memory cache with 5 minute TTL. If no valid info,
// executes `{runtime} images -q {tag}` to confirm image existence.
func (c *CapsuleCache) Exists() (bool, error) {
	c.mu.RLock()
	entry, ok := c.memCache[c.tag]
	if ok && timeNow().Sub(entry.timestamp) < cacheTTL {
		c.mu.RUnlock()
		return entry.exists, nil
	}
	c.mu.RUnlock()

	cmd := execCommand(c.runtime, "images", "-q", c.tag)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("%s images failed: %v", c.runtime, err)
	}
	exists := strings.TrimSpace(stdout.String()) != ""

	c.mu.Lock()
	c.memCache[c.tag] = cacheEntry{exists: exists, timestamp: timeNow()}
	c.mu.Unlock()

	return exists, nil
}

// ExistsWithDetails works like Exists but also returns image metadata
// using `{runtime} inspect {tag} --format='{{json .}}'`.
func (c *CapsuleCache) ExistsWithDetails() (bool, ImageDetails, error) {
	exists, err := c.Exists()
	if err != nil || !exists {
		return exists, ImageDetails{}, err
	}

	cmd := execCommand(c.runtime, "inspect", c.tag, "--format", "{{json .}}")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false, ImageDetails{}, fmt.Errorf("%s inspect failed: %v", c.runtime, err)
	}

	var details ImageDetails
	if err := json.Unmarshal(stdout.Bytes(), &details); err != nil {
		return true, ImageDetails{}, fmt.Errorf("parse inspect output: %w", err)
	}

	return true, details, nil
}

// InvalidateCache removes the cache entry to force re-verification
func (c *CapsuleCache) InvalidateCache() {
	c.mu.Lock()
	delete(c.memCache, c.tag)
	c.mu.Unlock()
}

// ValidateDigest checks that the image digest matches expected.
// This prevents using incorrect capsules when digest changes.
func (c *CapsuleCache) ValidateDigest(expectedDigest string) bool {
	exists, details, err := c.ExistsWithDetails()
	if err != nil || !exists {
		return false
	}
	for _, d := range details.RepoDigests {
		if strings.Contains(d, expectedDigest) {
			return true
		}
	}
	return false
}

// validateDigest provides a private version for backward compatibility
func (c *CapsuleCache) validateDigest(expectedDigest string) bool {
	return c.ValidateDigest(expectedDigest)
}

// Manager handles high-level cache operations
type Manager struct {
	runtime string
	stats   Statistics
	mu      sync.RWMutex
}

// NewManager creates a new cache manager
func NewManager(runtime string) *Manager {
	return &Manager{
		runtime: runtime,
	}
}

// GetCapsuleCache returns a CapsuleCache for a specific tag
func (m *Manager) GetCapsuleCache(tag string) *CapsuleCache {
	return NewCapsuleCache(m.runtime, tag)
}

// Stats returns cache statistics
func (m *Manager) Stats() Statistics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// ClearAll removes all cached images with mitl-capsule prefix
func (m *Manager) ClearAll() error {
	cmd := execCommand(m.runtime, "images", "--filter", "reference=mitl-capsule:*", "-q")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("list images failed: %w", err)
	}

	images := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, img := range images {
		if img != "" {
			_ = execCommand(m.runtime, "rmi", img).Run()
		}
	}

	return nil
}

// ClearOld removes capsules older than specified duration
func (m *Manager) ClearOld(age time.Duration) error {
	cutoff := timeNow().Add(-age).Format(time.RFC3339)
	cmd := execCommand(m.runtime, "images", "--filter", "reference=mitl-capsule:*",
		"--filter", fmt.Sprintf("before=%s", cutoff), "-q")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("list old images failed: %w", err)
	}

	images := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, img := range images {
		if img != "" {
			_ = execCommand(m.runtime, "rmi", img).Run()
		}
	}

	return nil
}
