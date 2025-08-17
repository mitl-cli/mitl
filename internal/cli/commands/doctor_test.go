package commands

import "testing"

func TestDoctor(t *testing.T) {
	if err := Doctor([]string{}); err != nil {
		t.Fatalf("doctor: %v", err)
	}
}
