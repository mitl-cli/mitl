package errors

import (
	stdErrors "errors"
	"strings"
	"testing"
)

func TestNewAndWrap(t *testing.T) {
	e := New(ErrBuildFailed, "Build failed")
	if e.Code != ErrBuildFailed || e.Message != "Build failed" {
		t.Fatalf("unexpected MitlError fields: %+v", e)
	}
	if e.Suggestion == "" {
		t.Error("expected default suggestion")
	}
	if len(e.Stack) == 0 {
		t.Error("expected stack frames captured")
	}
	if !strings.Contains(e.Error(), "Build failed") {
		t.Error("Error() should contain message")
	}

	// Wrap a std error
	base := stdErrors.New("boom")
	w := Wrap(base, ErrUnknown, "Something happened")
	if w.Cause == nil || !strings.Contains(w.Error(), "boom") {
		t.Error("wrapped error should include cause")
	}
}

func TestRecoverableAndContext(t *testing.T) {
	e := New(ErrRuntimeNotRunning, "runtime not running").WithContext("runtime", "docker")
	if !e.Recoverable {
		t.Error("ErrRuntimeNotRunning should be recoverable")
	}
	if e.Context["runtime"] != "docker" {
		t.Error("context key not set")
	}
}
