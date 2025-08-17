package exec

import (
	"os"
	"testing"
)

func TestFindBuildCLI_EnvOverride(t *testing.T) {
	old := os.Getenv("MITL_BUILD_CLI")
	defer os.Setenv("MITL_BUILD_CLI", old)
	os.Setenv("MITL_BUILD_CLI", "/bin/echo")
	got := FindBuildCLI()
	if got != "/bin/echo" {
		t.Fatalf("expected /bin/echo, got %q", got)
	}
}

func TestFindRunCLI_EnvOverride(t *testing.T) {
	old := os.Getenv("MITL_RUN_CLI")
	defer os.Setenv("MITL_RUN_CLI", old)
	os.Setenv("MITL_RUN_CLI", "/bin/echo")
	got := FindRunCLI()
	if got != "/bin/echo" {
		t.Fatalf("expected /bin/echo, got %q", got)
	}
}
