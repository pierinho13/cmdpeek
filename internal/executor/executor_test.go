package executor

import (
	"strings"
	"testing"
)

func TestRunReturnsErrorForFailingCommand(t *testing.T) {
	t.Parallel()

	err := Run("sh", "exit 7")
	if err == nil {
		t.Fatal("expected command execution error")
	}

	if !strings.Contains(err.Error(), "exit status 7") {
		t.Fatalf(
			"expected exit status in error, got %v",
			err,
		)
	}
}

func TestRunExecutesShellSyntax(t *testing.T) {
	t.Parallel()

	if err := Run("sh", "printf 'cmdpeek' >/dev/null"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunUsesConfiguredShell(t *testing.T) {
	t.Parallel()

	if err := Run(
		"bash",
		"set -euo pipefail; value=cmdpeek; [[ $value == cmdpeek ]]",
	); err != nil {
		t.Fatalf("Run() with bash error = %v", err)
	}
}
