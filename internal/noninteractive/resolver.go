package noninteractive

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pierinho13/cmdpeek/internal/catalog"
	variableresolver "github.com/pierinho13/cmdpeek/internal/variable"
)

func FindCommand(
	commands []catalog.Command,
	name string,
) (catalog.Command, error) {
	for _, command := range commands {
		if command.Name == name {
			return command, nil
		}
	}

	return catalog.Command{}, fmt.Errorf(
		"command %q was not found",
		name,
	)
}

func ResolveVariables(
	command catalog.Command,
	provided map[string]string,
	positional []string,
) (map[string]string, error) {
	values := make(map[string]string, len(command.Variables))
	knownVariables := make(
		map[string]struct{},
		len(command.Variables),
	)

	for _, variable := range command.Variables {
		knownVariables[variable.Name] = struct{}{}
	}

	unknown := make([]string, 0)
	for name := range provided {
		if _, exists := knownVariables[name]; !exists {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf(
			"unknown variables for command %q: %s",
			command.Name,
			strings.Join(unknown, ", "),
		)
	}

	positionalIndex := 0

	for _, variable := range command.Variables {
		value, exists := provided[variable.Name]

		if !exists && positionalIndex < len(positional) {
			value = positional[positionalIndex]
			positionalIndex++
			exists = true
		}

		resolvedValue, err := resolveVariable(
			command,
			variable,
			values,
			value,
			exists,
		)
		if err != nil {
			return nil, err
		}

		values[variable.Name] = resolvedValue
	}

	if positionalIndex < len(positional) {
		return nil, fmt.Errorf(
			"command %q received %d extra positional values",
			command.Name,
			len(positional)-positionalIndex,
		)
	}

	return values, nil
}

func resolveVariable(
	command catalog.Command,
	variable catalog.Variable,
	resolved map[string]string,
	providedValue string,
	hasProvidedValue bool,
) (string, error) {
	switch variable.Source.Type {
	case catalog.VariableSourceInput:
		if hasProvidedValue {
			return requireNonEmpty(variable, providedValue)
		}
		if variable.Default != "" {
			return variable.Default, nil
		}
		return "", missingValueError(variable)

	case catalog.VariableSourceEnvironment:
		if hasProvidedValue {
			return requireNonEmpty(variable, providedValue)
		}
		if value := strings.TrimSpace(
			os.Getenv(variable.Source.Variable),
		); value != "" {
			return value, nil
		}
		if variable.Default != "" {
			return variable.Default, nil
		}
		return "", missingValueError(variable)

	case catalog.VariableSourceOptions:
		value := providedValue
		if !hasProvidedValue {
			value = variable.Default
		}
		if strings.TrimSpace(value) == "" {
			return "", missingValueError(variable)
		}
		if !contains(variable.Source.Values, value) {
			return "", invalidOptionError(
				variable,
				value,
				variable.Source.Values,
			)
		}
		return value, nil

	case catalog.VariableSourceCommand:
		options, err := variableresolver.ResolveCommandOptions(
			command.Shell,
			variable.Source.Command,
			resolved,
		)
		if err != nil {
			return "", fmt.Errorf(
				"resolve variable %q: %w",
				variable.Name,
				err,
			)
		}

		value := providedValue
		if !hasProvidedValue {
			value = variable.Default
		}

		if strings.TrimSpace(value) != "" {
			if !contains(options, value) {
				return "", invalidOptionError(
					variable,
					value,
					options,
				)
			}
			return value, nil
		}

		if len(options) == 1 {
			return options[0], nil
		}

		return "", fmt.Errorf(
			"variable %q requires a value; available values: %s",
			variable.Name,
			strings.Join(options, ", "),
		)
	}

	return "", fmt.Errorf(
		"variable %q has unsupported source type %q",
		variable.Name,
		variable.Source.Type,
	)
}

func requireNonEmpty(
	variable catalog.Variable,
	value string,
) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", missingValueError(variable)
	}

	return value, nil
}

func missingValueError(variable catalog.Variable) error {
	return fmt.Errorf(
		"variable %q requires a value; provide it with --set %s=<value>",
		variable.Name,
		variable.Name,
	)
}

func invalidOptionError(
	variable catalog.Variable,
	value string,
	options []string,
) error {
	return fmt.Errorf(
		"invalid value %q for variable %q; available values: %s",
		value,
		variable.Name,
		strings.Join(options, ", "),
	)
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}
