package catalog

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v4"
)

var variableReferencePattern = regexp.MustCompile(
	`\{\{\s*([a-zA-Z_][a-zA-Z0-9_-]*)\s*\}\}`,
)

const (
	VariableSourceInput       = "input"
	VariableSourceOptions     = "options"
	VariableSourceEnvironment = "environment"
	VariableSourceCommand     = "command"
)

type Config struct {
	Version  int       `yaml:"version"`
	Shell    string    `yaml:"shell"`
	Commands []Command `yaml:"commands"`
}

type Command struct {
	Name        string     `yaml:"name"`
	Title       string     `yaml:"title"`
	Description string     `yaml:"description"`
	Labels      []string   `yaml:"labels"`
	Shell       string     `yaml:"shell"`
	Run         string     `yaml:"run"`
	Variables   []Variable `yaml:"variables"`
}

type Variable struct {
	Name        string         `yaml:"name"`
	Prompt      string         `yaml:"prompt"`
	Description string         `yaml:"description"`
	Default     string         `yaml:"default"`
	Source      VariableSource `yaml:"source"`
}

type VariableSource struct {
	Type     string   `yaml:"type"`
	Values   []string `yaml:"values"`
	Variable string   `yaml:"variable"`
	Command  string   `yaml:"command"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	if config.Version == 0 {
		config.Version = 1
	}

	if config.Version != 1 {
		return Config{}, fmt.Errorf(
			"unsupported config version %d",
			config.Version,
		)
	}

	config.Shell = strings.TrimSpace(config.Shell)
	if config.Shell == "" {
		config.Shell = "sh"
	}

	if len(config.Commands) == 0 {
		return Config{}, fmt.Errorf("config contains no commands")
	}

	names := make(map[string]struct{}, len(config.Commands))

	for index := range config.Commands {
		command := &config.Commands[index]

		command.Name = strings.TrimSpace(command.Name)
		command.Title = strings.TrimSpace(command.Title)
		command.Description = strings.TrimSpace(command.Description)
		command.Shell = strings.TrimSpace(command.Shell)
		command.Run = strings.TrimSpace(command.Run)

		if command.Name == "" {
			return Config{}, fmt.Errorf(
				"commands[%d].name is required",
				index,
			)
		}

		if command.Title == "" {
			command.Title = command.Name
		}

		if command.Shell == "" {
			command.Shell = config.Shell
		}

		if command.Run == "" {
			return Config{}, fmt.Errorf(
				"command %q has no run value",
				command.Name,
			)
		}

		if _, exists := names[command.Name]; exists {
			return Config{}, fmt.Errorf(
				"duplicate command name %q",
				command.Name,
			)
		}

		names[command.Name] = struct{}{}

		for labelIndex := range command.Labels {
			command.Labels[labelIndex] = strings.TrimSpace(
				command.Labels[labelIndex],
			)
		}

		if err := validateVariables(command); err != nil {
			return Config{}, err
		}
	}

	return config, nil
}

func validateVariables(command *Command) error {
	resolvedVariableNames := make(map[string]struct{}, len(command.Variables))

	for index := range command.Variables {
		variable := &command.Variables[index]

		variable.Name = strings.TrimSpace(variable.Name)
		variable.Prompt = strings.TrimSpace(variable.Prompt)
		variable.Description = strings.TrimSpace(variable.Description)
		variable.Default = strings.TrimSpace(variable.Default)
		variable.Source.Type = strings.TrimSpace(variable.Source.Type)
		variable.Source.Variable = strings.TrimSpace(
			variable.Source.Variable,
		)
		variable.Source.Command = strings.TrimSpace(
			variable.Source.Command,
		)

		if variable.Name == "" {
			return fmt.Errorf(
				"command %q variables[%d].name is required",
				command.Name,
				index,
			)
		}

		if _, exists := resolvedVariableNames[variable.Name]; exists {
			return fmt.Errorf(
				"command %q has duplicate variable %q",
				command.Name,
				variable.Name,
			)
		}

		if variable.Prompt == "" {
			variable.Prompt = variable.Name
		}

		switch variable.Source.Type {
		case VariableSourceInput:
		case VariableSourceOptions:
			if len(variable.Source.Values) == 0 {
				return fmt.Errorf(
					"command %q variable %q requires source.values",
					command.Name,
					variable.Name,
				)
			}

			for valueIndex := range variable.Source.Values {
				variable.Source.Values[valueIndex] = strings.TrimSpace(
					variable.Source.Values[valueIndex],
				)

				if variable.Source.Values[valueIndex] == "" {
					return fmt.Errorf(
						"command %q variable %q contains an empty option",
						command.Name,
						variable.Name,
					)
				}
			}

		case VariableSourceEnvironment:
			if variable.Source.Variable == "" {
				return fmt.Errorf(
					"command %q variable %q requires source.variable",
					command.Name,
					variable.Name,
				)
			}

		case VariableSourceCommand:
			if variable.Source.Command == "" {
				return fmt.Errorf(
					"command %q variable %q requires source.command",
					command.Name,
					variable.Name,
				)
			}

			for _, match := range variableReferencePattern.FindAllStringSubmatch(
				variable.Source.Command,
				-1,
			) {
				reference := match[1]

				if _, exists := resolvedVariableNames[reference]; !exists {
					return fmt.Errorf(
						"command %q variable %q references unresolved variable %q; command sources may only reference variables declared earlier",
						command.Name,
						variable.Name,
						reference,
					)
				}
			}

		default:
			return fmt.Errorf(
				"command %q variable %q has unsupported source type %q",
				command.Name,
				variable.Name,
				variable.Source.Type,
			)
		}

		resolvedVariableNames[variable.Name] = struct{}{}
	}

	return nil
}
