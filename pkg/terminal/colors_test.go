package terminal

import (
	"os"
	"testing"
)

func TestColorize_NoColorEnv(t *testing.T) {
	old := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer os.Setenv("NO_COLOR", old)

	txt := "hello"
	if got := Colorize(Red, txt); got != txt {
		t.Errorf("expected no colorization when NO_COLOR=1; got %q", got)
	}
	if got := BoldText(txt); got != txt {
		t.Errorf("expected no bold when NO_COLOR=1; got %q", got)
	}
}
