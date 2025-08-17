package commands

import (
	"os"
	"os/exec"
	"testing"
)

func withEnv(key, val string, fn func()) {
	old := os.Getenv(key)
	os.Setenv(key, val)
	defer os.Setenv(key, old)
	fn()
}

func TestCache_ListAndStats(t *testing.T) {
	withEnv("MITL_BUILD_CLI", "/bin/echo", func() {
		old := execCommand
		execCommand = func(name string, args ...string) *exec.Cmd {
			// Mimic docker images output
			return exec.Command("sh", "-c", "echo 'mitl-capsule abc' && true")
		}
		defer func() { execCommand = old }()

		if err := Cache([]string{"list"}); err != nil {
			t.Fatalf("list: %v", err)
		}

		// Stats path: images -q
		execCommand = func(name string, args ...string) *exec.Cmd {
			return exec.Command("sh", "-c", "echo 'id1' && echo 'id2'")
		}
		if err := Cache([]string{"stats"}); err != nil {
			t.Fatalf("stats: %v", err)
		}
	})
}

func TestCache_Clean(t *testing.T) {
	withEnv("MITL_BUILD_CLI", "/bin/echo", func() {
		old := execCommand
		// First call (images -q) returns ids; second (rmi ids...) succeeds
		calls := 0
		execCommand = func(name string, args ...string) *exec.Cmd {
			calls++
			if calls == 1 {
				return exec.Command("sh", "-c", "echo 'id1 id2'")
			}
			return exec.Command("sh", "-c", "true")
		}
		defer func() { execCommand = old }()
		if err := Cache([]string{"clean"}); err != nil {
			t.Fatalf("clean: %v", err)
		}
	})
}
