package main

import (
	"os"
	"testing"
)

func TestMain_NoArgs(t *testing.T) {
	old := os.Args
	os.Args = []string{"mitl"}
	defer func() { os.Args = old }()
	main()
}

func TestMain_Version(t *testing.T) {
	old := os.Args
	os.Args = []string{"mitl", "version"}
	defer func() { os.Args = old }()
	main()
}
