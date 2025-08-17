package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	e "mitl/pkg/errors"
)

func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	_ = w.Close()
	os.Stdout = old
	var b strings.Builder
	_, _ = io.Copy(&b, r)
	return b.String()
}

func TestErrorHandler_DisplayMitlError(t *testing.T) {
	h := NewErrorHandler(true, false) // verbose
	err := e.New(e.ErrBuildFailed, "Build failed").
		WithDetails("Invalid syntax at line 10").
		WithSuggestion("Run mitl doctor").
		WithContext("file", "Dockerfile")

	out := captureStdout(t, func() {
		h.displayMitlError(err)
	})
	if !strings.Contains(out, "Build failed") || !strings.Contains(out, "Invalid syntax") {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "Dockerfile") || !strings.Contains(out, "mitl doctor") {
		t.Fatalf("missing context/suggestion: %s", out)
	}
}
