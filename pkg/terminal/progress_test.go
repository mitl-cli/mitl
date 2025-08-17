package terminal

import "testing"

func TestProgressBar_NonTerminal(t *testing.T) {
	p := NewProgressBar(3, "test")
	p.Update(1)
	p.Increment()
	p.Finish()
}
