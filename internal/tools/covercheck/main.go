package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type fileCov struct {
	total   int
	covered int
}

func main() {
	var profile string
	var threshold float64
	var include string
	flag.StringVar(&profile, "profile", "coverage.out", "coverage profile file (go test -coverprofile)")
	flag.Float64Var(&threshold, "threshold", 85.0, "minimum per-file coverage percentage")
	flag.StringVar(&include, "include", "", "comma-separated path prefixes to include (optional)")
	flag.Parse()

	f, err := os.Open(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "covercheck: failed to open profile: %v\n", err)
		os.Exit(2)
	}
	defer f.Close()

	cov := make(map[string]*fileCov)
	var filters []string
	if include != "" {
		parts := strings.Split(include, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				filters = append(filters, filepath.ToSlash(p))
			}
		}
	}
	s := bufio.NewScanner(f)
	first := true
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if first {
			first = false
			// header like: mode: set/count/atomic
			continue
		}
		if line == "" {
			continue
		}
		// Format: filename.go:startLine.startCol,endLine.endCol numStatements count
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		fileAndRange := fields[0]
		numStmtStr := fields[1]
		countStr := fields[2]

		// Extract filename before colon
		i := strings.Index(fileAndRange, ":")
		if i <= 0 {
			continue
		}
		filename := fileAndRange[:i]
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}
		// Normalize to repo-root relative
		filename = filepath.ToSlash(filename)
		if len(filters) > 0 {
			ok := false
			for _, f := range filters {
				if strings.HasPrefix(filename, f) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}

		numStmt, err1 := strconv.Atoi(numStmtStr)
		cnt, err2 := strconv.Atoi(countStr)
		if err1 != nil || err2 != nil {
			continue
		}
		fc := cov[filename]
		if fc == nil {
			fc = &fileCov{}
			cov[filename] = fc
		}
		fc.total += numStmt
		if cnt > 0 {
			fc.covered += numStmt
		}
	}
	if err := s.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "covercheck: read error: %v\n", err)
		os.Exit(2)
	}

	// Evaluate
	var failed []string
	for file, fc := range cov {
		if fc.total == 0 {
			// no statements; ignore
			continue
		}
		pct := float64(fc.covered) * 100.0 / float64(fc.total)
		if pct+1e-9 < threshold { // guard for float errors
			failed = append(failed, fmt.Sprintf("%s: %.1f%% < %.1f%%", file, pct, threshold))
		}
	}

	if len(failed) > 0 {
		fmt.Fprintln(os.Stderr, "Per-file coverage check failed:")
		for _, msg := range failed {
			fmt.Fprintln(os.Stderr, "  ", msg)
		}
		os.Exit(1)
	}
}
