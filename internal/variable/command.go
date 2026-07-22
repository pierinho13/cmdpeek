package variable

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	commandtemplate "github.com/pierinho13/cmdpeek/internal/template"
)

const commandTimeout = 10 * time.Second

func ResolveCommandOptions(
	shell string,
	command string,
	values map[string]string,
) ([]string, error) {
	renderedCommand, err := commandtemplate.Render(command, values)
	if err != nil {
		return nil, fmt.Errorf(
			"render variable source command: %w",
			err,
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		commandTimeout,
	)
	defer cancel()

	if shell == "" {
		shell = "sh"
	}

	process := exec.CommandContext(
		ctx,
		shell,
		"-c",
		renderedCommand,
	)

	output, err := process.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf(
			"variable source command timed out after %s",
			commandTimeout,
		)
	}

	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}

		return nil, fmt.Errorf(
			"variable source command failed: %s",
			message,
		)
	}

	options := parseOptions(string(output))
	if len(options) == 0 {
		return nil, fmt.Errorf(
			"variable source command returned no options",
		)
	}

	return options, nil
}

func parseOptions(output string) []string {
	lines := strings.Split(output, "\n")
	seen := make(map[string]struct{}, len(lines))
	options := make([]string, 0, len(lines))

	for _, line := range lines {
		option := strings.TrimSpace(line)
		if option == "" {
			continue
		}

		if _, exists := seen[option]; exists {
			continue
		}

		seen[option] = struct{}{}
		options = append(options, option)
	}

	return options
}
