package commands

import (
	"os"
	"os/exec"
	"testing"
)

func TestVolumes_StatsAndClean(t *testing.T) {
	os.Setenv("MITL_RUN_CLI", "/bin/echo")
	old := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd { return exec.Command("sh", "-c", "true") }
	defer func() { execCommand = old }()
	_ = Volumes([]string{"stats"})
	_ = Volumes([]string{"clean", "1"})
}
