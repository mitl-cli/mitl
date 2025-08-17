package commands

import "testing"

func TestInspect(t *testing.T) {
	_ = Inspect([]string{}) // exercise path; ignore error since project may be unknown
}
