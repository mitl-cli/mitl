package commands

import "testing"

func TestRuntime_Info(t *testing.T) {
	_ = Runtime([]string{"info"})
}

func TestRuntime_Recommend(t *testing.T) {
	_ = Runtime([]string{"recommend"})
}
