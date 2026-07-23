package noninteractive

import (
	"strings"
	"testing"

	"github.com/pierinho13/cmdpeek/internal/catalog"
)

func TestFindCommandUsesExactName(t *testing.T) {
	t.Parallel()

	commands := []catalog.Command{
		{Name: "gcommit"},
		{Name: "deploy"},
	}

	command, err := FindCommand(commands, "gcommit")
	if err != nil {
		t.Fatalf("FindCommand() error = %v", err)
	}

	if command.Name != "gcommit" {
		t.Fatalf("expected gcommit, got %q", command.Name)
	}
}

func TestResolveVariablesUsesNamedValuesAndDefaults(t *testing.T) {
	t.Parallel()

	command := catalog.Command{
		Name: "deploy",
		Variables: []catalog.Variable{
			{
				Name:    "environment",
				Default: "staging",
				Source: catalog.VariableSource{
					Type:   catalog.VariableSourceOptions,
					Values: []string{"staging", "production"},
				},
			},
			{
				Name: "version",
				Source: catalog.VariableSource{
					Type: catalog.VariableSourceInput,
				},
			},
		},
	}

	values, err := ResolveVariables(
		command,
		map[string]string{"version": "1.2.3"},
		nil,
	)
	if err != nil {
		t.Fatalf("ResolveVariables() error = %v", err)
	}

	if values["environment"] != "staging" {
		t.Fatalf(
			"expected staging, got %q",
			values["environment"],
		)
	}
	if values["version"] != "1.2.3" {
		t.Fatalf(
			"expected 1.2.3, got %q",
			values["version"],
		)
	}
}

func TestResolveVariablesSupportsPositionalValues(t *testing.T) {
	t.Parallel()

	command := catalog.Command{
		Name: "gcommit",
		Variables: []catalog.Variable{
			{
				Name: "message",
				Source: catalog.VariableSource{
					Type: catalog.VariableSourceInput,
				},
			},
			{
				Name: "branch",
				Source: catalog.VariableSource{
					Type: catalog.VariableSourceInput,
				},
			},
		},
	}

	values, err := ResolveVariables(
		command,
		nil,
		[]string{"add feature", "main"},
	)
	if err != nil {
		t.Fatalf("ResolveVariables() error = %v", err)
	}

	if values["message"] != "add feature" ||
		values["branch"] != "main" {
		t.Fatalf("unexpected values: %#v", values)
	}
}

func TestResolveVariablesRejectsInvalidOption(t *testing.T) {
	t.Parallel()

	command := catalog.Command{
		Name: "deploy",
		Variables: []catalog.Variable{
			{
				Name: "environment",
				Source: catalog.VariableSource{
					Type:   catalog.VariableSourceOptions,
					Values: []string{"staging", "production"},
				},
			},
		},
	}

	_, err := ResolveVariables(
		command,
		map[string]string{"environment": "test"},
		nil,
	)
	if err == nil {
		t.Fatal("expected invalid option error")
	}

	if !strings.Contains(err.Error(), "available values") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveVariablesRejectsUnknownNamedValue(t *testing.T) {
	t.Parallel()

	command := catalog.Command{Name: "hello"}

	_, err := ResolveVariables(
		command,
		map[string]string{"unknown": "value"},
		nil,
	)
	if err == nil {
		t.Fatal("expected unknown variable error")
	}
}
