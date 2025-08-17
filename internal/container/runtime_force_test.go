package container

import "testing"

func TestRuntime_ForceBenchmarkAndDescriptions(t *testing.T) {
	rm := NewManager()
	rm.availableRuntimes = []Runtime{{Name: "echo", Path: "/bin/echo"}}
	rm.ForceBenchmark(false)
	rm.ForceBenchmark(true)

	// cover runtimeDescription cases
	_ = runtimeDescription("container")
	_ = runtimeDescription("finch")
	_ = runtimeDescription("docker")
	_ = runtimeDescription("podman")
	_ = runtimeDescription("nerdctl")
	_ = runtimeDescription("unknown")
}
