// Package cli provides the command-line interface for the mitl tool.
// It implements a modular command system with support for subcommands,
// help text, and version information. The CLI uses a registry pattern
// to register available commands and route execution based on user input.
//
// The main components are:
//   - CLI: The main interface that handles command routing and execution
//   - Command: Interface that all commands must implement
//   - Command registry: Maps command names to their implementations
//
// Commands are implemented in the commands subpackage and registered
// during CLI initialization for clean separation of concerns.
package cli

import (
	"fmt"

	"mitl/internal/config"
	"mitl/pkg/version"
)

// Command represents a CLI command
type Command interface {
	Name() string
	Description() string
	Run(args []string) error
}

// CLI represents the command-line interface
type CLI struct {
	config   *config.Config
	commands map[string]Command
}

// New creates a new CLI instance
func New(cfg *config.Config) *CLI {
	c := &CLI{config: cfg, commands: make(map[string]Command)}
	c.registerCommands()
	return c
}

func (c *CLI) register(cmd Command) {
	c.commands[cmd.Name()] = cmd
}

// registerCommands registers all available commands
func (c *CLI) registerCommands() {
	c.register(NewAnalyzeCommand())
	c.register(NewDigestCommand())
	c.register(NewHydrateCommand())
	c.register(NewBuildCommand())
	c.register(NewRunCommand())
	c.register(NewShellCommand())
	c.register(NewInspectCommand())
	c.register(NewSetupCommand())
	c.register(NewRuntimeCommand())
	c.register(NewDoctorCommand())
	c.register(NewCacheCommand())
	c.register(NewVolumesCommand())
	c.register(NewBenchCommand())
	c.register(NewCompletionCommand())
}

// Run executes the CLI with given arguments
func (c *CLI) Run(args []string) error {
	if len(args) < 2 {
		c.printUsage()
		return nil
	}
	switch args[1] {
	case "help", "--help", "-h":
		c.printUsage()
		return nil
	case "version", "--version", "-v":
		fmt.Printf("mitl %s\n", version.Version)
		return nil
	default:
		// Try registered commands
		if cmd, ok := c.commands[args[1]]; ok {
			return cmd.Run(args[2:])
		}
		c.printUsage()
		return fmt.Errorf("unknown command: %s", args[1])
	}
}

func (c *CLI) printUsage() {
	fmt.Println("Usage: mitl <command> [args]")
	fmt.Println("Commands:")
	for name, cmd := range c.commands {
		fmt.Printf("  %-8s %s\n", name, cmd.Description())
	}
	fmt.Println("  version  Show version")
	fmt.Println("  help     Show this help")
}
