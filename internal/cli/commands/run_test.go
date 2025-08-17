package commands

import "testing"

func TestRun_NoArgs(t *testing.T) {
	if err := Run(nil); err == nil {
		t.Fatalf("expected error when no args provided")
	}
}
