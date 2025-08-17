// Package cli: Central error handling for CLI
// Provides consistent error presentation, recovery attempts, and suggestions
package cli

import (
	"fmt"
	"os"
	"strings"

	e "mitl/pkg/errors"
	"mitl/pkg/terminal"
)

// ErrorHandler handles errors consistently across the CLI
type ErrorHandler struct {
	verbose   bool
	debug     bool
	noColor   bool
	recoverer *e.Recoverer
}

// NewErrorHandler creates an error handler
func NewErrorHandler(verbose, debug bool) *ErrorHandler {
	return &ErrorHandler{
		verbose:   verbose,
		debug:     debug,
		recoverer: e.NewRecoverer(verbose),
	}
}

// Handle processes an error and displays it to the user
func (h *ErrorHandler) Handle(err error) {
	if err == nil {
		return
	}

	if mitlErr, ok := err.(*e.MitlError); ok {
		if mitlErr.Recoverable {
			if recErr := h.recoverer.Recover(mitlErr); recErr == nil {
				// Recovered; treat as success
				return
			}
		}
		h.displayMitlError(mitlErr)
	} else {
		// Wrap unknown
		mitlErr := e.Wrap(err, e.ErrUnknown, "An unexpected error occurred")
		h.displayMitlError(mitlErr)
	}
	os.Exit(1)
}

func (h *ErrorHandler) displayMitlError(err *e.MitlError) {
	fmt.Println()
	icon := h.getErrorIcon(err.Code)
	fmt.Printf("%s %s%s%s\n", icon, terminal.Bold, err.Message, terminal.Reset)

	if err.Details != "" && h.verbose {
		fmt.Printf("\n%s%s%s\n", terminal.Dim, err.Details, terminal.Reset)
	}

	if len(err.Context) > 0 && h.verbose {
		fmt.Println("\nContext:")
		for k, v := range err.Context {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	if err.Suggestion != "" {
		fmt.Printf("\n💡 %s%s%s\n", terminal.Yellow, err.Suggestion, terminal.Reset)
	}

	if err.Cause != nil && h.verbose {
		fmt.Printf("\n%sCaused by:%s\n", terminal.Dim, terminal.Reset)
		h.displayCauseChain(err.Cause, 1)
	}

	if h.debug && len(err.Stack) > 0 {
		fmt.Printf("\n%sStack trace:%s\n", terminal.Dim, terminal.Reset)
		for _, f := range err.Stack {
			fmt.Printf("  %s\n", h.formatStackFrame(f))
		}
	}

	fmt.Println()
	if !h.verbose {
		fmt.Printf("%sRun with --verbose for more details%s\n", terminal.Dim, terminal.Reset)
	}
	if !h.debug && err.Code == e.ErrUnknown {
		fmt.Printf("%sRun with --debug for stack trace%s\n", terminal.Dim, terminal.Reset)
	}
}

func (h *ErrorHandler) displayCauseChain(err error, depth int) {
	indent := strings.Repeat("  ", depth)
	if mitlErr, ok := err.(*e.MitlError); ok {
		fmt.Printf("%s• %s\n", indent, mitlErr.Message)
		if mitlErr.Cause != nil {
			h.displayCauseChain(mitlErr.Cause, depth+1)
		}
		return
	}
	fmt.Printf("%s• %s\n", indent, err.Error())
}

func (h *ErrorHandler) formatStackFrame(frame e.StackFrame) string {
	file := frame.File
	if idx := strings.LastIndex(file, "/mitl/"); idx >= 0 {
		file = "..." + file[idx:]
	}
	fn := frame.Function
	if idx := strings.LastIndex(fn, "."); idx >= 0 {
		fn = fn[idx+1:]
	}
	return fmt.Sprintf("%s:%d %s()", file, frame.Line, fn)
}

func (h *ErrorHandler) getErrorIcon(code e.ErrorCode) string {
	icons := map[e.ErrorCode]string{
		e.ErrRuntimeNotFound:    "🔍",
		e.ErrRuntimeNotRunning:  "💤",
		e.ErrRuntimePermission:  "🔒",
		e.ErrBuildFailed:        "❌",
		e.ErrDockerfileNotFound: "📄",
		e.ErrCacheCorrupted:     "💔",
		e.ErrDiskFull:           "💾",
		e.ErrFileNotFound:       "🔍",
		e.ErrPermissionDenied:   "🚫",
		e.ErrNetworkTimeout:     "🌐",
		e.ErrInvalidConfig:      "⚙️",
		e.ErrUnknown:            "❓",
	}
	if ic, ok := icons[code]; ok {
		return ic
	}
	return "❌"
}
