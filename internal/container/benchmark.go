package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// needsBenchmark returns true when cache is missing, stale, or incomplete
func (m *Manager) needsBenchmark() bool {
	if os.Getenv("MITL_NO_BENCHMARK") == "1" {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.availableRuntimes) == 0 {
		return false
	}

	info, err := os.Stat(m.configPath)
	if err != nil {
		return true
	}
	if time.Since(info.ModTime()) > benchTTL {
		return true
	}

	f, err := os.ReadFile(m.configPath)
	if err != nil {
		return true
	}
	var cf benchmarkCacheFile
	if json.Unmarshal(f, &cf) != nil {
		return true
	}
	if cf.Hardware.OS != m.hardwareProfile.OS ||
		cf.Hardware.Arch != m.hardwareProfile.Arch ||
		cf.Hardware.IsAppleSilicon != m.hardwareProfile.IsAppleSilicon {
		return true
	}

	for _, rt := range m.availableRuntimes {
		if _, ok := cf.Results[rt.Name]; !ok {
			return true
		}
	}
	return false
}

// benchmarkAll performs a simple build+run test for each runtime
func (m *Manager) benchmarkAll(includeBuild bool) {
	results := make([]BenchmarkResult, 0, len(m.availableRuntimes))
	for _, rt := range m.availableRuntimes {
		res := m.benchmarkRuntime(rt, includeBuild)
		if includeBuild {
			res.Mode = "build+exec"
		} else {
			res.Mode = "exec"
		}
		results = append(results, res)
	}
	m.normalizeScores(results)
	m.saveBenchmarkCache(results)
	m.updateRuntimeScores(results)

	successes := 0
	for _, r := range results {
		if r.Error == "" && r.Score > 0 {
			successes++
		}
	}
	if successes == 0 {
		img := os.Getenv("MITL_BENCH_IMAGE")
		if img == "" {
			img = "alpine:latest"
		}
		fmt.Printf("Benchmark could not run (likely no local images or network blocked). Pre-pull '%s' and retry.\n", img)
	}
}

func (m *Manager) benchmarkRuntime(rt Runtime, includeBuild bool) BenchmarkResult {
	result := BenchmarkResult{Runtime: rt.Name, Timestamp: time.Now()}
	benchImage := os.Getenv("MITL_BENCH_IMAGE")
	if benchImage == "" {
		benchImage = "alpine:latest"
	}

	var testTag string
	if includeBuild {
		tmpDir, _ := os.MkdirTemp("", "mitl-bench-")
		defer os.RemoveAll(tmpDir)
		dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
		content := fmt.Sprintf("FROM %s\nRUN echo benchmark\nCMD [\"echo\", \"hello\"]\n", benchImage)
		_ = os.WriteFile(dockerfilePath, []byte(content), 0644)

		testTag = fmt.Sprintf("mitl-bench-%s-%d", rt.Name, time.Now().UnixNano())

		buildStart := time.Now()
		buildCmd := execCommand(rt.Path, "build", "-t", testTag, "-f", dockerfilePath, tmpDir)
		buildErr := buildCmd.Run()
		result.BuildTime = time.Since(buildStart)
		if buildErr != nil {
			result.Error = fmt.Sprintf("build failed: %v", buildErr)
			testTag = ""
		}
	}

	imageToTest := benchImage
	if testTag != "" {
		imageToTest = testTag
		defer func() {
			_ = execCommand(rt.Path, "rmi", testTag).Run()
		}()
	}

	startStart := time.Now()
	containerName := fmt.Sprintf("mitl-bench-%d", time.Now().UnixNano())
	runCmd := execCommand(rt.Path, "run", "--rm", "--name", containerName, imageToTest, "echo", "hello")
	output, runErr := runCmd.Output()
	totalTime := time.Since(startStart)

	if runErr != nil {
		result.Error = fmt.Sprintf("run failed: %v", runErr)
		return result
	}
	if strings.TrimSpace(string(output)) != "hello" {
		result.Error = "unexpected output"
		return result
	}

	result.StartTime = totalTime / 2
	result.ExecTime = totalTime / 2

	if includeBuild {
		result.Score = result.BuildTime.Seconds() + result.StartTime.Seconds()
	} else {
		result.Score = result.StartTime.Seconds() + result.ExecTime.Seconds()
	}

	return result
}

// normalizeScores converts absolute scores to relative (fastest=1.0)
func (m *Manager) normalizeScores(results []BenchmarkResult) {
	min := 0.0
	for _, r := range results {
		if r.Error != "" {
			continue
		}
		if min == 0 || r.Score < min {
			min = r.Score
		}
	}
	if min <= 0 {
		// no successful results
		return
	}
	for i := range results {
		if results[i].Error == "" && results[i].Score > 0 {
			results[i].Score = results[i].Score / min
		}
	}
}

// saveBenchmarkCache persists results to disk
func (m *Manager) saveBenchmarkCache(results []BenchmarkResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cache := benchmarkCacheFile{
		Hardware: m.hardwareProfile,
		Results:  make(map[string]BenchmarkResult),
	}

	for _, r := range results {
		cache.Results[r.Runtime] = r
		m.benchmarkCache[r.Runtime] = r
	}

	data, _ := json.MarshalIndent(cache, "", "  ")
	_ = os.WriteFile(m.configPath, data, 0644)
}

// loadBenchmarkCache reads cached benchmark results
func (m *Manager) loadBenchmarkCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return
	}

	var cache benchmarkCacheFile
	if json.Unmarshal(data, &cache) != nil {
		return
	}

	for k, v := range cache.Results {
		m.benchmarkCache[k] = v
	}
}

// updateRuntimeScores applies benchmark results to runtime Performance field
func (m *Manager) updateRuntimeScores(results []BenchmarkResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.availableRuntimes {
		for _, r := range results {
			if m.availableRuntimes[i].Name == r.Runtime && r.Error == "" {
				m.availableRuntimes[i].Performance = r.Score
				m.availableRuntimes[i].LastBenchmark = r.Timestamp
			}
		}
	}
}

// applyCachedPerformance applies cached benchmark scores to runtimes
func (m *Manager) applyCachedPerformance() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.availableRuntimes {
		if cached, ok := m.benchmarkCache[m.availableRuntimes[i].Name]; ok {
			if cached.Error == "" && time.Since(cached.Timestamp) < benchTTL {
				m.availableRuntimes[i].Performance = cached.Score
				m.availableRuntimes[i].LastBenchmark = cached.Timestamp
			}
		}
	}
}

// selectByPerformance returns the best runtime name by lowest normalized score
func (m *Manager) selectByPerformance() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	best := ""
	bestScore := 0.0
	for _, rt := range m.availableRuntimes {
		res, ok := m.benchmarkCache[rt.Name]
		if !ok || res.Error != "" || res.Score <= 0 {
			continue
		}
		if best == "" || res.Score < bestScore {
			best = rt.Name
			bestScore = res.Score
		}
	}
	return best
}

// getRelativeSpeed returns how many times faster the chosen runtime is vs the slowest of others
func (m *Manager) getRelativeSpeed(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	chosen, ok := m.benchmarkCache[name]
	if !ok || chosen.Score <= 0 {
		return 0
	}
	max := 0.0
	for n, r := range m.benchmarkCache {
		if n == name || r.Error != "" || r.Score <= 0 {
			continue
		}
		if r.Score > max {
			max = r.Score
		}
	}
	if max <= 0 {
		return 0
	}
	// both chosen and others are normalized to fastest=1, so max/chosen is a ratio
	return max / chosen.Score
}
