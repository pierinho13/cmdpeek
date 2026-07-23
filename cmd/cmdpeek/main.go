package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/pierinho13/cmdpeek/internal/catalog"
	"github.com/pierinho13/cmdpeek/internal/configsource"
	"github.com/pierinho13/cmdpeek/internal/executor"
	"github.com/pierinho13/cmdpeek/internal/noninteractive"
	commandtemplate "github.com/pierinho13/cmdpeek/internal/template"
	"github.com/pierinho13/cmdpeek/internal/tui"
)

type setFlags struct {
	values map[string]string
}

func (flags *setFlags) String() string {
	if flags == nil || len(flags.values) == 0 {
		return ""
	}

	pairs := make([]string, 0, len(flags.values))
	for name, value := range flags.values {
		pairs = append(pairs, name+"="+value)
	}

	return strings.Join(pairs, ",")
}

func (flags *setFlags) Set(value string) error {
	name, variableValue, found := strings.Cut(value, "=")
	name = strings.TrimSpace(name)

	if !found || name == "" {
		return fmt.Errorf("expected NAME=VALUE")
	}

	if flags.values == nil {
		flags.values = make(map[string]string)
	}

	if _, exists := flags.values[name]; exists {
		return fmt.Errorf("variable %q was provided more than once", name)
	}

	flags.values[name] = variableValue
	return nil
}

func main() {
	configFlag := flag.String(
		"config",
		"",
		"path to the cmdpeek configuration file",
	)
	noInteractiveFlag := flag.Bool(
		"no-interactive",
		false,
		"select a command and resolve variables without the interactive catalog",
	)
	nameFlag := flag.String(
		"name",
		"",
		"exact command name used with --no-interactive",
	)
	yesFlag := flag.Bool(
		"yes",
		false,
		"execute without asking for confirmation",
	)
	dryRunFlag := flag.Bool(
		"dry-run",
		false,
		"render and print the command without executing it",
	)

	var providedValues setFlags
	flag.Var(
		&providedValues,
		"set",
		"set a command variable using NAME=VALUE; may be repeated",
	)

	flag.Parse()

	if !*noInteractiveFlag {
		if strings.TrimSpace(*nameFlag) != "" ||
			len(providedValues.values) > 0 ||
			*yesFlag {
			exitWithError(
				fmt.Errorf(
					"--name, --set and --yes require --no-interactive",
				),
			)
		}
	}

	configPath, warning, err := configsource.Resolve(*configFlag)
	if err != nil {
		exitWithError(err)
	}

	if warning != "" {
		fmt.Fprintf(os.Stderr, "cmdpeek: warning: %s\n", warning)
	}

	config, err := catalog.Load(configPath)
	if err != nil {
		exitWithError(err)
	}

	if *noInteractiveFlag {
		runNonInteractive(
			config,
			*nameFlag,
			providedValues.values,
			flag.Args(),
			*yesFlag,
			*dryRunFlag,
		)
		return
	}

	runInteractive(
		config,
		strings.Join(flag.Args(), " "),
		*dryRunFlag,
	)
}

func runInteractive(
	config catalog.Config,
	initialQuery string,
	dryRun bool,
) {
	selected, err := tui.Run(config.Commands, initialQuery)
	if err != nil {
		exitWithError(err)
	}

	if selected == nil {
		return
	}

	values, err := tui.ResolveVariables(*selected)
	if err != nil {
		exitWithError(err)
	}

	renderedCommand, err := commandtemplate.Render(selected.Run, values)
	if err != nil {
		exitWithError(err)
	}

	if dryRun {
		printRenderedCommand(renderedCommand)
		return
	}

	confirmed, err := tui.ConfirmExecution(renderedCommand)
	if err != nil {
		exitWithError(err)
	}

	if !confirmed {
		return
	}

	execute(*selected, renderedCommand)
}

func runNonInteractive(
	config catalog.Config,
	commandName string,
	providedValues map[string]string,
	positionalValues []string,
	yes bool,
	dryRun bool,
) {
	commandName = strings.TrimSpace(commandName)
	if commandName == "" {
		exitWithError(
			fmt.Errorf("--name is required with --no-interactive"),
		)
	}

	selected, err := noninteractive.FindCommand(
		config.Commands,
		commandName,
	)
	if err != nil {
		exitWithError(err)
	}

	values, err := noninteractive.ResolveVariables(
		selected,
		providedValues,
		positionalValues,
	)
	if err != nil {
		exitWithError(err)
	}

	renderedCommand, err := commandtemplate.Render(
		selected.Run,
		values,
	)
	if err != nil {
		exitWithError(err)
	}

	if dryRun {
		printRenderedCommand(renderedCommand)
		return
	}

	if !yes {
		confirmed, err := tui.ConfirmExecution(renderedCommand)
		if err != nil {
			exitWithError(err)
		}
		if !confirmed {
			return
		}
	}

	execute(selected, renderedCommand)
}

func execute(command catalog.Command, renderedCommand string) {
	printRenderedCommand(renderedCommand)

	if err := executor.Run(command.Shell, renderedCommand); err != nil {
		exitWithError(err)
	}
}

func printRenderedCommand(command string) {
	fmt.Printf("\n$ %s\n\n", command)
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "cmdpeek: %v\n", err)
	os.Exit(1)
}
