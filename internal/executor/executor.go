package executor

import (
	"fmt"
	"os"
	"os/exec"
)

// Run executes the rendered command using the user's shell environment.
//
// The command is intentionally executed through /bin/sh because cmdpeek
// commands may contain pipes, redirects, environment assignments and other
// shell syntax.
func Run(shell string, command string) error {
	if shell == "" {
		shell = "sh"
	}

	process := exec.Command(shell, "-c", command)

	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	process.Env = os.Environ()

	if err := process.Run(); err != nil {
		return fmt.Errorf(
			"execute command %q: %w",
			command,
			err,
		)
	}

	return nil
}
