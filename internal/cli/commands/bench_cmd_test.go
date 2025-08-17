package commands

import "testing"

func TestBench_HelpAndList(t *testing.T) {
	cmd := NewBenchCommand()
	_ = cmd.Run([]string{"help"}) // exercise usage
	_ = cmd.Run([]string{"list"})
}
