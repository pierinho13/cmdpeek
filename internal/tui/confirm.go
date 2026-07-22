package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	confirmationQuestionStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("170"))

	confirmationCommandStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("252")).
					Padding(1, 2).
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("240"))

	confirmationChoiceStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))
)

type confirmationModel struct {
	command   string
	confirmed bool
	answered  bool
}

func newConfirmationModel(command string) confirmationModel {
	return confirmationModel{
		command: command,
	}
}

func (model confirmationModel) Init() tea.Cmd {
	return nil
}

func (model confirmationModel) Update(
	message tea.Msg,
) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyPressMsg:
		switch strings.ToLower(message.String()) {
		case "y", "yes":
			model.confirmed = true
			model.answered = true
			return model, tea.Quit

		case "n", "no", "enter", "esc", "q", "ctrl+c":
			model.confirmed = false
			model.answered = true
			return model, tea.Quit
		}
	}

	return model, nil
}

func (model confirmationModel) View() tea.View {
	var builder strings.Builder

	builder.WriteString("\n  ")
	builder.WriteString(titleStyle.Render("cmdpeek"))
	builder.WriteString("\n\n  ")
	builder.WriteString(
		confirmationQuestionStyle.Render(
			"Execute this command?",
		),
	)
	builder.WriteString("\n\n")

	builder.WriteString(
		confirmationCommandStyle.Render(model.command),
	)
	builder.WriteString("\n\n  ")

	builder.WriteString(
		confirmationChoiceStyle.Render(
			"y execute   n cancel   enter cancel",
		),
	)
	builder.WriteString("\n")

	return tea.NewView(builder.String())
}

// ConfirmExecution asks the user for an explicit confirmation before running
// the rendered command. The default answer is no.
func ConfirmExecution(command string) (bool, error) {
	if strings.TrimSpace(command) == "" {
		return false, fmt.Errorf(
			"cannot confirm an empty command",
		)
	}

	finalModel, err := tea.NewProgram(
		newConfirmationModel(command),
	).Run()
	if err != nil {
		return false, fmt.Errorf(
			"run command confirmation: %w",
			err,
		)
	}

	model, ok := finalModel.(confirmationModel)
	if !ok {
		return false, fmt.Errorf(
			"unexpected confirmation model type",
		)
	}

	if !model.answered {
		return false, nil
	}

	return model.confirmed, nil
}
