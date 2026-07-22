package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pierinho13/cmdpeek/internal/catalog"
	commandtemplate "github.com/pierinho13/cmdpeek/internal/template"
	variableresolver "github.com/pierinho13/cmdpeek/internal/variable"
)

const variablePreviewLines = 8

var (
	variablePromptStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("170"))

	variableDescriptionStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("244"))

	variableFieldLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("245")).
				Width(13)

	variableSourceStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170"))

	variableProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	optionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	selectedOptionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("170"))

	variablePreviewLabelStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("245"))

	variablePreviewStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250"))

	variableErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))
)

type variableModel struct {
	command catalog.Command

	index           int
	optionIndex     int
	currentOptions  []string
	values          map[string]string
	input           textinput.Model
	optionFilter    textinput.Model
	optionFiltering bool
	errMessage      string
	previewLines    int
	previewOffset   int
	width           int
	height          int

	cancelled bool
	completed bool
}

func newVariableModel(command catalog.Command) variableModel {
	model := variableModel{
		command: command,
		values:  make(map[string]string, len(command.Variables)),
		width:   defaultWidth,
		height:  defaultHeight,
	}

	model.optionFilter = textinput.New()
	model.optionFilter.Prompt = "Search: "
	model.optionFilter.CharLimit = 200

	model.prepareCurrentVariable()
	model.autoAdvanceSingleCommandOptions()

	return model
}

func (model variableModel) Init() tea.Cmd {
	if model.currentVariableUsesInput() || model.optionFiltering {
		return textinput.Blink
	}

	return nil
}

func (model variableModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		model.width = message.Width
		model.height = message.Height
		model.clampPreviewWindow()
		return model, nil

	case tea.KeyPressMsg:
		if message.String() == "ctrl+c" {
			model.cancelled = true
			return model, tea.Quit
		}

		if model.optionFiltering {
			switch message.String() {
			case "esc":
				model.optionFilter.SetValue("")
				model.optionFiltering = false
				model.optionFilter.Blur()
				model.optionIndex = 0
				return model, nil

			case "enter":
				model.optionFiltering = false
				model.optionFilter.Blur()
				return model, nil
			}

			var command tea.Cmd
			model.optionFilter, command = model.optionFilter.Update(message)
			model.optionIndex = 0
			return model, command
		}

		switch message.String() {
		case "esc":
			model.cancelled = true
			return model, tea.Quit

		case "ctrl+up":
			model.scrollPreview(-1)
			return model, nil

		case "ctrl+down":
			model.scrollPreview(1)
			return model, nil
		}

		if model.index >= len(model.command.Variables) {
			return model, nil
		}

		variable := model.command.Variables[model.index]

		if model.currentVariableIsOptions() {
			switch message.String() {
			case "/":
				model.optionFiltering = true
				model.optionFilter.Focus()
				return model, textinput.Blink

			case "up", "k":
				model.moveOption(-1)
				return model, nil

			case "down", "j":
				model.moveOption(1)
				return model, nil

			case "left":
				model.moveOptionPage(-1)
				return model, nil

			case "right":
				model.moveOptionPage(1)
				return model, nil

			case "r":
				if variable.Source.Type == catalog.VariableSourceCommand {
					model.prepareCurrentVariable()
				}
				return model, nil

			case "enter":
				options := model.filteredOptions()
				if len(options) == 0 {
					return model, nil
				}

				model.values[variable.Name] = options[model.optionIndex]

				return model.advance()
			}

			return model, nil
		}

		if message.String() == "enter" {
			value := strings.TrimSpace(model.input.Value())

			if value == "" {
				model.errMessage = fmt.Sprintf(
					"%s cannot be empty",
					variable.Prompt,
				)
				return model, nil
			}

			model.values[variable.Name] = value

			return model.advance()
		}
	}

	if model.currentVariableUsesInput() {
		var command tea.Cmd
		model.input, command = model.input.Update(message)
		return model, command
	}

	return model, nil
}

func (model variableModel) View() tea.View {
	var builder strings.Builder

	builder.WriteString("\n  ")
	builder.WriteString(titleStyle.Render("cmdpeek"))
	builder.WriteString("\n  ")
	builder.WriteString(
		variableProgressStyle.Render(
			fmt.Sprintf(
				"%s · variable %d of %d",
				model.command.Title,
				model.index+1,
				len(model.command.Variables),
			),
		),
	)
	builder.WriteString("\n\n")

	if model.index < len(model.command.Variables) {
		variable := model.command.Variables[model.index]

		builder.WriteString("  ")
		builder.WriteString(
			variablePromptStyle.Render("Provide command variable"),
		)
		builder.WriteString("\n\n")

		builder.WriteString(
			renderVariableField("Variable", variable.Name),
		)
		builder.WriteString(
			renderVariableField(
				"Type",
				variableSourceStyle.Render(variable.Source.Type),
			),
		)
		builder.WriteString(
			renderVariableField("Prompt", variable.Prompt),
		)

		if variable.Description != "" {
			builder.WriteString(
				renderVariableField(
					"Description",
					variableDescriptionStyle.Render(
						variable.Description,
					),
				),
			)
		}

		builder.WriteString("\n")

		if model.currentVariableIsOptions() {
			builder.WriteString(model.renderOptions(variable))
		} else {
			builder.WriteString("  ")
			builder.WriteString(
				variableFieldLabelStyle.Render("Value"),
			)
			builder.WriteString(model.input.View())
			builder.WriteString("\n")
		}
	}

	if model.errMessage != "" {
		builder.WriteString("\n  ")
		builder.WriteString(
			variableErrorStyle.Render(model.errMessage),
		)
		builder.WriteString("\n")
	}

	builder.WriteString("\n  ")
	builder.WriteString(
		variablePreviewLabelStyle.Render("Command preview"),
	)
	builder.WriteString("\n")

	preview, _, _ := model.previewWindow()

	previewWidth := model.previewBoxWidth()

	previewBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(previewWidth).
		MaxWidth(previewWidth).
		Render(preview)

	builder.WriteString("  ")
	builder.WriteString(previewBox)
	builder.WriteString("\n")

	totalLines := len(model.previewCommandLines())

	startLine := model.previewOffset + 1
	endLine := model.previewOffset + model.previewViewportLines()
	if endLine > totalLines {
		endLine = totalLines
	}

	builder.WriteString("  ")
	builder.WriteString(
		variableProgressStyle.Render(
			fmt.Sprintf(
				"lines %d-%d of %d",
				startLine,
				endLine,
				totalLines,
			),
		),
	)
	builder.WriteString("\n")

	builder.WriteString("\n")

	if model.currentVariableIsOptions() {
		help := "  / search   ↑/↓ navigate   ←/→ page   enter select   ctrl+↑/↓ preview   esc cancel"

		if model.currentVariableIsCommand() {
			help = "  / search   ↑/↓ navigate   ←/→ page   enter select   r retry   ctrl+↑/↓ preview   esc cancel"
		}

		builder.WriteString(
			helpStyle.Render(help),
		)
	} else {
		builder.WriteString(
			helpStyle.Render(
				"  type value   ctrl+↑/↓ scroll preview   enter continue   esc cancel",
			),
		)
	}

	builder.WriteString("\n")

	view := tea.NewView(builder.String())
	view.AltScreen = true

	return view
}

func (model *variableModel) scrollPreview(delta int) {
	model.previewOffset += delta
	model.clampPreviewWindow()
}

func (model variableModel) maxPreviewOffset() int {
	total := len(model.previewCommandLines())
	maximum := total - model.previewViewportLines()
	if maximum < 0 {
		return 0
	}

	return maximum
}

func (model *variableModel) clampPreviewWindow() {
	maximumOffset := model.maxPreviewOffset()

	if model.previewOffset > maximumOffset {
		model.previewOffset = maximumOffset
	}

	if model.previewOffset < 0 {
		model.previewOffset = 0
	}
}

func (model variableModel) previewBoxWidth() int {
	// Leave room for the two-space indentation, border and terminal margin.
	width := model.width - 8
	if width < 30 {
		width = 30
	}

	return width
}

func (model variableModel) previewViewportLines() int {
	maximum := model.height - 15
	if maximum < 1 {
		maximum = 1
	}

	if maximum < variablePreviewLines {
		return maximum
	}

	return variablePreviewLines
}

func (model variableModel) previewCommandLines() []string {
	preview := commandtemplate.Preview(
		model.command.Run,
		model.previewValues(),
	)

	preview = strings.Trim(preview, "\n")
	if preview == "" {
		return []string{""}
	}

	return strings.Split(preview, "\n")
}

func renderVariableField(label string, value string) string {
	return "  " +
		variableFieldLabelStyle.Render(label) +
		value +
		"\n"
}

func (model variableModel) previewValues() map[string]string {
	values := make(
		map[string]string,
		len(model.values)+1,
	)

	for name, value := range model.values {
		values[name] = value
	}

	if model.index >= len(model.command.Variables) {
		return values
	}

	variable := model.command.Variables[model.index]

	if model.currentVariableUsesInput() {
		if value := model.input.Value(); value != "" {
			values[variable.Name] = value
		}

		return values
	}

	if model.currentVariableIsOptions() {
		options := model.filteredOptions()
		if len(options) > 0 && model.optionIndex < len(options) {
			values[variable.Name] = options[model.optionIndex]
		}
	}

	return values
}

func (model variableModel) previewWindow() (string, int, int) {
	lines := model.previewCommandLines()

	start := model.previewOffset
	if start > len(lines) {
		start = len(lines)
	}

	end := start + model.previewViewportLines()
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[start:end]
	hiddenAbove := start
	hiddenBelow := len(lines) - end

	return indentPreviewLines(visible), hiddenAbove, hiddenBelow
}

func indentPreviewLines(lines []string) string {
	result := make([]string, len(lines))

	for index, line := range lines {
		result[index] = "  " + line
	}

	return strings.Join(result, "\n")
}

func (model variableModel) renderOptions(
	variable catalog.Variable,
) string {
	var builder strings.Builder

	if model.optionFiltering || model.optionFilter.Value() != "" {
		builder.WriteString("  ")
		builder.WriteString(model.optionFilter.View())
		builder.WriteString("\n\n")
	} else {
		builder.WriteString("  ")
		builder.WriteString(
			variableProgressStyle.Render(
				"Search: press / to filter options",
			),
		)
		builder.WriteString("\n\n")
	}

	options := model.filteredOptions()
	if len(options) == 0 {
		builder.WriteString("  ")
		builder.WriteString(
			variableDescriptionStyle.Render("No matching options"),
		)
		builder.WriteString("\n")
		return builder.String()
	}

	pageSize := model.optionPageSize()
	page := model.optionIndex / pageSize
	start := page * pageSize
	end := start + pageSize
	if end > len(options) {
		end = len(options)
	}

	for index := start; index < end; index++ {
		value := options[index]

		if index == model.optionIndex {
			builder.WriteString(
				selectedMarkerStyle.Render("▌ "),
			)
			builder.WriteString(
				selectedOptionStyle.Render(value),
			)
		} else {
			builder.WriteString("  ")
			builder.WriteString(optionStyle.Render(value))
		}

		builder.WriteString("\n")
	}

	totalPages := (len(options) + pageSize - 1) / pageSize
	builder.WriteString("\n  ")
	builder.WriteString(
		variableProgressStyle.Render(
			fmt.Sprintf(
				"Page %d/%d · %d options",
				page+1,
				totalPages,
				len(options),
			),
		),
	)
	builder.WriteString("\n")

	return builder.String()
}

func (model variableModel) filteredOptions() []string {
	query := strings.ToLower(strings.TrimSpace(model.optionFilter.Value()))
	if query == "" {
		return model.currentOptions
	}

	filtered := make([]string, 0, len(model.currentOptions))
	for _, value := range model.currentOptions {
		if strings.Contains(strings.ToLower(value), query) {
			filtered = append(filtered, value)
		}
	}

	return filtered
}

func (model variableModel) optionPageSize() int {
	size := model.height - 28
	if size < 5 {
		return 5
	}
	if size > 12 {
		return 12
	}

	return size
}

func (model *variableModel) moveOption(delta int) {
	optionCount := len(model.filteredOptions())

	if optionCount == 0 {
		model.optionIndex = 0
		return
	}

	model.optionIndex += delta

	if model.optionIndex < 0 {
		model.optionIndex = optionCount - 1
	}

	if model.optionIndex >= optionCount {
		model.optionIndex = 0
	}
}

func (model *variableModel) moveOptionPage(delta int) {
	options := model.filteredOptions()
	if len(options) == 0 {
		model.optionIndex = 0
		return
	}

	pageSize := model.optionPageSize()
	currentPage := model.optionIndex / pageSize
	totalPages := (len(options) + pageSize - 1) / pageSize

	nextPage := currentPage + delta
	if nextPage < 0 {
		nextPage = totalPages - 1
	}
	if nextPage >= totalPages {
		nextPage = 0
	}

	offset := model.optionIndex % pageSize
	model.optionIndex = nextPage*pageSize + offset
	if model.optionIndex >= len(options) {
		model.optionIndex = len(options) - 1
	}
}

func (model variableModel) advance() (tea.Model, tea.Cmd) {
	model.index++
	model.optionIndex = 0
	model.errMessage = ""

	if model.index >= len(model.command.Variables) {
		model.completed = true
		return model, tea.Quit
	}

	model.prepareCurrentVariable()
	model.autoAdvanceSingleCommandOptions()

	if model.completed {
		return model, tea.Quit
	}

	if model.currentVariableUsesInput() {
		return model, textinput.Blink
	}

	return model, nil
}

func (model *variableModel) prepareCurrentVariable() {
	if model.index >= len(model.command.Variables) {
		return
	}

	variable := model.command.Variables[model.index]

	model.currentOptions = nil
	model.optionIndex = 0
	model.optionFilter.SetValue("")
	model.optionFiltering = false
	model.optionFilter.Blur()
	model.errMessage = ""

	if variable.Source.Type == catalog.VariableSourceOptions {
		model.currentOptions = append(
			[]string(nil),
			variable.Source.Values...,
		)
		model.optionIndex = defaultOptionIndex(
			variable,
			model.currentOptions,
		)
		return
	}

	if variable.Source.Type == catalog.VariableSourceCommand {
		options, err := variableresolver.ResolveCommandOptions(
			model.command.Shell,
			variable.Source.Command,
			model.values,
		)
		if err != nil {
			model.errMessage = err.Error()
			return
		}

		model.currentOptions = options
		model.optionIndex = defaultOptionIndex(
			variable,
			model.currentOptions,
		)
		return
	}

	model.input = textinput.New()
	model.input.Prompt = "> "
	model.input.CharLimit = 500

	initialValue := variable.Default

	if variable.Source.Type == catalog.VariableSourceEnvironment {
		if environmentValue := os.Getenv(
			variable.Source.Variable,
		); environmentValue != "" {
			initialValue = environmentValue
		}
	}

	model.input.SetValue(initialValue)
	model.input.Focus()
}

func (model *variableModel) autoAdvanceSingleCommandOptions() {
	for model.index < len(model.command.Variables) {
		if !model.currentVariableIsCommand() ||
			model.errMessage != "" ||
			len(model.currentOptions) != 1 {
			return
		}

		variable := model.command.Variables[model.index]
		model.values[variable.Name] = model.currentOptions[0]

		model.index++
		model.optionIndex = 0
		model.currentOptions = nil
		model.errMessage = ""

		if model.index >= len(model.command.Variables) {
			model.completed = true
			return
		}

		model.prepareCurrentVariable()
	}
}

func defaultOptionIndex(
	variable catalog.Variable,
	options []string,
) int {
	if variable.Default == "" {
		return 0
	}

	for index, value := range options {
		if value == variable.Default {
			return index
		}
	}

	return 0
}

func (model variableModel) currentVariableUsesInput() bool {
	if model.index >= len(model.command.Variables) {
		return false
	}

	sourceType := model.command.Variables[model.index].Source.Type

	return sourceType == catalog.VariableSourceInput ||
		sourceType == catalog.VariableSourceEnvironment
}

func (model variableModel) currentVariableIsOptions() bool {
	if model.index >= len(model.command.Variables) {
		return false
	}

	sourceType := model.command.Variables[model.index].Source.Type

	return sourceType == catalog.VariableSourceOptions ||
		sourceType == catalog.VariableSourceCommand
}

func (model variableModel) currentVariableIsCommand() bool {
	if model.index >= len(model.command.Variables) {
		return false
	}

	return model.command.Variables[model.index].Source.Type ==
		catalog.VariableSourceCommand
}

func ResolveVariables(
	command catalog.Command,
) (map[string]string, error) {
	if len(command.Variables) == 0 {
		return map[string]string{}, nil
	}

	initialModel := newVariableModel(command)
	if initialModel.completed {
		return initialModel.values, nil
	}

	program := tea.NewProgram(initialModel)

	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("resolve variables: %w", err)
	}

	model, ok := finalModel.(variableModel)
	if !ok {
		return nil, fmt.Errorf(
			"unexpected variable model type",
		)
	}

	if model.cancelled || !model.completed {
		return nil, fmt.Errorf("variable resolution cancelled")
	}

	return model.values, nil
}
