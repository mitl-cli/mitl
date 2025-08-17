package commands

import (
	"mitl/internal/doctor"
)

// Doctor runs system health checks and diagnostics.
// Supports flags: --verbose, --fix
func Doctor(args []string) error {
	verbose := false
	fix := false
	for _, a := range args {
		switch a {
		case "--verbose", "-v":
			verbose = true
		case "--fix":
			fix = true
		}
	}
	doctor.RunDoctorWithOptions(verbose, fix)
	return nil
}
