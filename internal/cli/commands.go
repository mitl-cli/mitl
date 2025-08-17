package cli

import (
	"mitl/internal/cli/commands"
)

// Real command implementations using the extracted command functions
type analyzeCmd struct{}

func (analyzeCmd) Name() string        { return "analyze" }
func (analyzeCmd) Description() string { return "Analyze host toolchains" }
func (analyzeCmd) Run(args []string) error {
	return commands.Analyze(args)
}

type hydrateCmd struct{}

func (hydrateCmd) Name() string        { return "hydrate" }
func (hydrateCmd) Description() string { return "Build project capsule" }
func (hydrateCmd) Run(args []string) error {
	return commands.Hydrate(args)
}

type runCmd struct{}

func (runCmd) Name() string        { return "run" }
func (runCmd) Description() string { return "Run command in capsule" }
func (runCmd) Run(args []string) error {
	return commands.Run(args)
}

type shellCmd struct{}

func (shellCmd) Name() string        { return "shell" }
func (shellCmd) Description() string { return "Open shell in capsule" }
func (shellCmd) Run(args []string) error {
	return commands.Shell(args)
}

type setupCmd struct{}

func (setupCmd) Name() string        { return "setup" }
func (setupCmd) Description() string { return "Setup default runtime" }
func (setupCmd) Run(args []string) error {
	return commands.Setup(args)
}

type inspectCmd struct{}

func (inspectCmd) Name() string        { return "inspect" }
func (inspectCmd) Description() string { return "Analyze project and show Dockerfile" }
func (inspectCmd) Run(args []string) error {
	return commands.Inspect(args)
}

type cacheCmd struct{}

func (cacheCmd) Name() string        { return "cache" }
func (cacheCmd) Description() string { return "Cache management" }
func (cacheCmd) Run(args []string) error {
	return commands.Cache(args)
}

type volumesCmd struct{}

func (volumesCmd) Name() string        { return "volumes" }
func (volumesCmd) Description() string { return "Volume management" }
func (volumesCmd) Run(args []string) error {
	return commands.Volumes(args)
}

type runtimeCmd struct{}

func (runtimeCmd) Name() string        { return "runtime" }
func (runtimeCmd) Description() string { return "Runtime info/benchmark/recommend" }
func (runtimeCmd) Run(args []string) error {
	return commands.Runtime(args)
}

// Doctor command - this one was already properly implemented
type doctorCmd struct{}

func (doctorCmd) Name() string        { return "doctor" }
func (doctorCmd) Description() string { return "System health check" }
func (doctorCmd) Run(args []string) error {
	return commands.Doctor(args)
}

// Bench command implementation
type benchCmd struct{}

func (benchCmd) Name() string        { return "bench" }
func (benchCmd) Description() string { return "Run benchmarks and performance comparisons" }
func (benchCmd) Run(args []string) error {
	cmd := commands.NewBenchCommand()
	return cmd.Run(args)
}

// Command factory functions
func NewAnalyzeCommand() Command { return analyzeCmd{} }

// Digest command implementation
type digestCmd struct{}

func (digestCmd) Name() string        { return "digest" }
func (digestCmd) Description() string { return "Calculate and inspect project digests" }
func (digestCmd) Run(args []string) error {
	return commands.Digest(args)
}

func NewDigestCommand() Command  { return digestCmd{} }
func NewHydrateCommand() Command { return hydrateCmd{} }
func NewRunCommand() Command     { return runCmd{} }
func NewShellCommand() Command   { return shellCmd{} }
func NewInspectCommand() Command { return inspectCmd{} }
func NewSetupCommand() Command   { return setupCmd{} }
func NewRuntimeCommand() Command { return runtimeCmd{} }
func NewDoctorCommand() Command  { return doctorCmd{} }
func NewCacheCommand() Command   { return cacheCmd{} }
func NewVolumesCommand() Command { return volumesCmd{} }
func NewBenchCommand() Command   { return benchCmd{} }

// Completion command implementation
type completionCmd struct{}

func (completionCmd) Name() string        { return "completion" }
func (completionCmd) Description() string { return "Generate shell completion scripts" }
func (completionCmd) Run(args []string) error {
	return commands.Completion(args)
}

func NewCompletionCommand() Command { return completionCmd{} }

// build command alias for hydrate
type buildCmd struct{}

func (buildCmd) Name() string            { return "build" }
func (buildCmd) Description() string     { return "Build project capsule (alias for hydrate)" }
func (buildCmd) Run(args []string) error { return commands.Hydrate(args) }

func NewBuildCommand() Command { return buildCmd{} }
