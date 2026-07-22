package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pierinho13/cmdpeek/internal/catalog"
	"github.com/pierinho13/cmdpeek/internal/executor"
	commandtemplate "github.com/pierinho13/cmdpeek/internal/template"
	"github.com/pierinho13/cmdpeek/internal/tui"
)

const (
	configEnvironmentVariable = "CMD_PEEK_CONFIG_FILE"
	defaultConfigPath         = ".cmdpeek.yaml"
)

func main() {
	configFlag := flag.String(
		"config",
		"",
		"path to the cmdpeek configuration file",
	)

	flag.Parse()

	configPath := resolveConfigPath(*configFlag)

	config, err := catalog.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmdpeek: %v\n", err)
		os.Exit(1)
	}

	selected, err := tui.Run(config.Commands)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmdpeek: %v\n", err)
		os.Exit(1)
	}

	if selected == nil {
		return
	}

	values, err := tui.ResolveVariables(*selected)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmdpeek: %v\n", err)
		os.Exit(1)
	}

	renderedCommand, err := commandtemplate.Render(selected.Run, values)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmdpeek: %v\n", err)
		os.Exit(1)
	}

	confirmed, err := tui.ConfirmExecution(renderedCommand)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmdpeek: %v\n", err)
		os.Exit(1)
	}

	if !confirmed {
		return
	}

	fmt.Printf("\n$ %s\n\n", renderedCommand)

	if err := executor.Run(selected.Shell, renderedCommand); err != nil {
		fmt.Fprintf(os.Stderr, "cmdpeek: %v\n", err)
		os.Exit(1)
	}
}

func resolveConfigPath(configFlag string) string {
	if configFlag != "" {
		return configFlag
	}

	if configPath := os.Getenv(configEnvironmentVariable); configPath != "" {
		return configPath
	}

	return defaultConfigPath
}
