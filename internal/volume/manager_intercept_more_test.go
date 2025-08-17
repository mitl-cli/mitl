package volume

import (
	"strings"
	"testing"
)

func TestManager_InterceptNodeCommand_More(t *testing.T) {
	vm := NewManager("true", t.TempDir())
	// npm test -> pnpm test
	out := vm.InterceptNodeCommand([]string{"npm", "test"})
	s := sliceToString(out)
	if !strings.Contains(s, "pnpm test") {
		t.Fatalf("expected pnpm test mapping, got %s", s)
	}
	// yarn install -> pnpm install
	out = vm.InterceptNodeCommand([]string{"yarn", "install"})
	s = sliceToString(out)
	if !strings.Contains(s, "pnpm install") {
		t.Fatalf("expected pnpm install mapping, got %s", s)
	}
}

func sliceToString(s []string) string {
	if len(s) == 0 {
		return ""
	}
	out := s[0]
	for i := 1; i < len(s); i++ {
		out += " " + s[i]
	}
	return out
}
