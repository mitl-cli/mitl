package commands

import (
	"fmt"
	"os"

	"mitl/internal/digest"

	e "mitl/pkg/errors"
)

// Shell opens an interactive shell inside the capsule Docker image.
func Shell(args []string) error {
	// Use deterministic project digest for capsule tag
	digestValue, derr := digest.ProjectTag(".", digest.Options{Algorithm: "sha256"})
	if derr != nil {
		return e.Wrap(derr, e.ErrUnknown, "Failed to compute project digest").
			WithSuggestion("Run 'mitl digest --verbose' for details")
	}
	tag := fmt.Sprintf("mitl-capsule:%s", digestValue)
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cli := findRunCLI()
	containerArgs := []string{"run", "-it", "--rm", "-v", fmt.Sprintf("%s:/app", cwd), "-w", "/app", tag, "/bin/bash"}
	cmd := execCommand(cli, containerArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
