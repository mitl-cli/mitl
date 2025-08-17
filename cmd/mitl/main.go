package main

import (
	"os"
	"strings"

	"mitl/internal/cli"
	"mitl/internal/config"
	"mitl/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// Fall back to running with nil config; handle errors centrally
		cfg = nil
	}

	// Parse global flags (lightweight) and strip them from args
	verbose := false
	debug := false
	args := make([]string, 0, len(os.Args))
	for i, a := range os.Args {
		if i == 0 {
			args = append(args, a)
			continue
		}
		switch a {
		case "--verbose":
			verbose = true
		case "--debug":
			debug = true
		default:
			// keep normal args
			args = append(args, a)
		}
	}
	// Env overrides
	if strings.EqualFold(os.Getenv("MITL_VERBOSE"), "1") {
		verbose = true
	}
	if strings.EqualFold(os.Getenv("MITL_DEBUG"), "1") {
		debug = true
	}

	// Initialize logging
	logger.Initialize(verbose, debug)
	defer logger.Close()

	handler := cli.NewErrorHandler(verbose, debug)
	// Install a panic recoverer to avoid raw panics
	var ph cli.PanicHandler
	defer ph.Recover()

	app := cli.New(cfg)
	if err := app.Run(args); err != nil {
		handler.Handle(err)
	}
}
