// Package errors provides enhanced error types with context and recovery
// metadata for mitl. These errors carry suggestions, context map, and
// lightweight stack traces to improve user diagnostics and recovery.
package errors

import (
	"runtime"
	"strings"
)

// ErrorCode categorizes errors for handling
type ErrorCode string

const (
	// Runtime errors
	ErrRuntimeNotFound   ErrorCode = "RUNTIME_NOT_FOUND"
	ErrRuntimeNotRunning ErrorCode = "RUNTIME_NOT_RUNNING"
	ErrRuntimePermission ErrorCode = "RUNTIME_PERMISSION"

	// Build errors
	ErrBuildFailed        ErrorCode = "BUILD_FAILED"
	ErrDockerfileNotFound ErrorCode = "DOCKERFILE_NOT_FOUND"
	ErrInvalidDockerfile  ErrorCode = "INVALID_DOCKERFILE"

	// Cache errors
	ErrCacheCorrupted  ErrorCode = "CACHE_CORRUPTED"
	ErrCachePermission ErrorCode = "CACHE_PERMISSION"

	// Filesystem errors
	ErrDiskFull         ErrorCode = "DISK_FULL"
	ErrFileNotFound     ErrorCode = "FILE_NOT_FOUND"
	ErrPermissionDenied ErrorCode = "PERMISSION_DENIED"

	// Network errors
	ErrNetworkTimeout      ErrorCode = "NETWORK_TIMEOUT"
	ErrRegistryUnreachable ErrorCode = "REGISTRY_UNREACHABLE"

	// Configuration errors
	ErrInvalidConfig ErrorCode = "INVALID_CONFIG"
	ErrMissingConfig ErrorCode = "MISSING_CONFIG"

	// Unknown errors
	ErrUnknown ErrorCode = "UNKNOWN"
)

// StackFrame represents a single stack frame
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// MitlError is the base error type with rich context
type MitlError struct {
	Code        ErrorCode         `json:"code"`
	Message     string            `json:"message"`
	Details     string            `json:"details,omitempty"`
	Suggestion  string            `json:"suggestion,omitempty"`
	Cause       error             `json:"-"`
	Context     map[string]string `json:"context,omitempty"`
	Recoverable bool              `json:"recoverable"`
	Stack       []StackFrame      `json:"stack,omitempty"`
}

// Error implements the error interface
func (e *MitlError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Message)
	if e.Details != "" {
		sb.WriteString("\n")
		sb.WriteString(e.Details)
	}
	if e.Cause != nil {
		sb.WriteString("\nCaused by: ")
		sb.WriteString(e.Cause.Error())
	}
	return sb.String()
}

// WithSuggestion adds a suggestion for fixing the error
func (e *MitlError) WithSuggestion(suggestion string) *MitlError {
	e.Suggestion = suggestion
	return e
}

// WithContext adds contextual information
func (e *MitlError) WithContext(key, value string) *MitlError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// WithCause wraps another error
func (e *MitlError) WithCause(cause error) *MitlError {
	e.Cause = cause
	return e
}

// WithDetails adds detailed information
func (e *MitlError) WithDetails(details string) *MitlError {
	e.Details = details
	return e
}

// New creates a new MitlError
func New(code ErrorCode, message string) *MitlError {
	err := &MitlError{
		Code:        code,
		Message:     message,
		Recoverable: isRecoverable(code),
		Context:     make(map[string]string),
	}
	err.captureStack()
	err.Suggestion = getDefaultSuggestion(code)
	return err
}

// Wrap wraps a standard error with MitlError
func Wrap(err error, code ErrorCode, message string) *MitlError {
	if err == nil {
		return nil
	}
	if mitlErr, ok := err.(*MitlError); ok {
		// Prepend message context
		if message != "" {
			mitlErr.Message = message + ": " + mitlErr.Message
		}
		return mitlErr
	}
	return New(code, message).WithCause(err)
}

// captureStack captures the current stack trace
func (e *MitlError) captureStack() {
	const maxFrames = 10
	pc := make([]uintptr, maxFrames)
	n := runtime.Callers(3, pc) // Skip runtime.Callers, captureStack, New/Wrap
	frames := runtime.CallersFrames(pc[:n])
	for {
		frame, more := frames.Next()
		if strings.Contains(frame.File, "runtime/") || strings.Contains(frame.File, "testing/") {
			if !more {
				break
			}
			continue
		}
		e.Stack = append(e.Stack, StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})
		if !more {
			break
		}
	}
}

// isRecoverable determines if an error can be automatically recovered
func isRecoverable(code ErrorCode) bool {
    switch code {
    case ErrRuntimeNotRunning, ErrCacheCorrupted, ErrRegistryUnreachable, ErrNetworkTimeout:
        return true
    case ErrRuntimeNotFound,
        ErrRuntimePermission,
        ErrBuildFailed,
        ErrDockerfileNotFound,
        ErrInvalidDockerfile,
        ErrCachePermission,
        ErrDiskFull,
        ErrFileNotFound,
        ErrPermissionDenied,
        ErrInvalidConfig,
        ErrMissingConfig,
        ErrUnknown:
        return false
    default:
        return false
    }
}

// getDefaultSuggestion provides default fix suggestions
func getDefaultSuggestion(code ErrorCode) string {
	suggestions := map[ErrorCode]string{
		ErrRuntimeNotFound:    "Install Docker or Podman: brew install --cask docker",
		ErrRuntimeNotRunning:  "Start Docker: open -a Docker",
		ErrRuntimePermission:  "Fix permissions: sudo usermod -aG docker $USER",
		ErrBuildFailed:        "Check Dockerfile syntax and try: mitl doctor",
		ErrDockerfileNotFound: "Generate Dockerfile: mitl init",
		ErrDiskFull:           "Free disk space: mitl cache clean",
		ErrPermissionDenied:   "Check file permissions or run with sudo",
		ErrNetworkTimeout:     "Check internet connection and retry",
		ErrInvalidConfig:      "Fix config: mitl config validate",
	}
	if s, ok := suggestions[code]; ok {
		return s
	}
	return "Run 'mitl doctor' for diagnostics"
}
