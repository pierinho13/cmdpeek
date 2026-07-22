package tui

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pierinho13/cmdpeek/internal/catalog"
)

const (
	defaultWidth  = 100
	defaultHeight = 30

	normalItemHeight       = 1
	selectedBaseHeight     = 7
	variableSummaryLines   = 3
	defaultCommandPageSize = 5
)

var variablePattern = regexp.MustCompile(
	`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`,
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	countStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	normalCommandStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	selectedCommandStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("170"))

	selectedMarkerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("170"))

	detailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("245")).
				Width(13)

	descriptionValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250"))

	nameValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	labelsValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	commandValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	variableStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	variablesHeadingStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("245"))

	variableNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("250"))

	variableTypeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170"))

	variableMetadataStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	noResultsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	commandTypeSummaryStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
)

type filteredCommand struct {
	command catalog.Command
	score   int
}

type Model struct {
	commands []catalog.Command
	filtered []catalog.Command

	filterInput textinput.Model
	filtering   bool

	cursor   int
	offset   int
	page     int
	pageSize int

	width  int
	height int

	selected    catalog.Command
	hasSelected bool

	detailMode   bool
	detailOffset int
}

func New(commands []catalog.Command) Model {
	input := textinput.New()
	input.Prompt = "Search: "
	input.Placeholder = "title, description, label or command"
	input.CharLimit = 200

	model := Model{
		commands:    commands,
		filtered:    commands,
		filterInput: input,
		pageSize:    defaultCommandPageSize,
		width:       defaultWidth,
		height:      defaultHeight,
	}

	model.clampCursor()

	return model
}

func (model Model) Init() tea.Cmd {
	return nil
}

func (model Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		model.width = message.Width
		model.height = message.Height

		if model.detailMode {
			model.clampDetailOffset()
		} else {
			model.ensureCursorVisible()
		}

		return model, nil

	case tea.KeyPressMsg:
		if model.detailMode {
			return model.updateDetail(message)
		}

		if model.filtering {
			return model.updateFiltering(message)
		}

		switch message.String() {
		case "q", "ctrl+c":
			return model, tea.Quit

		case "/":
			model.filtering = true
			model.filterInput.Focus()
			return model, textinput.Blink

		case "up", "k":
			model.moveCursor(-1)

		case "down", "j":
			model.moveCursor(1)

		case "left":
			model.previousPage()

		case "right":
			model.nextPage()

		case "home", "g":
			model.cursor = 0
			model.page = 0
			model.ensureCursorVisible()

		case "end", "G":
			if len(model.filtered) > 0 {
				model.cursor = len(model.filtered) - 1
				model.page = model.cursor / model.pageSize
				model.ensureCursorVisible()
			}

		case "esc":
			if model.filterInput.Value() != "" {
				model.filterInput.SetValue("")
				model.applyFilter()
				model.cursor = 0
				model.offset = 0
				model.page = 0
				model.ensureCursorVisible()
			}

		case "e":
			if len(model.filtered) > 0 {
				model.detailMode = true
				model.detailOffset = 0
			}

		case "enter":
			return model.selectCurrentCommand()
		}
	}

	return model, nil
}

func (model Model) updateDetail(
	message tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q", "ctrl+c":
		return model, tea.Quit

	case "esc", "b", "e":
		model.detailMode = false
		model.detailOffset = 0
		return model, nil

	case "up", "k":
		if model.detailOffset > 0 {
			model.detailOffset--
		}

	case "down", "j":
		model.detailOffset++
		model.clampDetailOffset()

	case "left":
		model.detailOffset -= model.detailViewportHeight()
		if model.detailOffset < 0 {
			model.detailOffset = 0
		}

	case "right":
		model.detailOffset += model.detailViewportHeight()
		model.clampDetailOffset()

	case "home", "g":
		model.detailOffset = 0

	case "end", "G":
		model.detailOffset = model.maxDetailOffset()

	case "enter":
		return model.selectCurrentCommand()
	}

	return model, nil
}

func (model Model) updateFiltering(
	message tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "ctrl+c":
		return model, tea.Quit

	case "esc":
		model.filtering = false
		model.filterInput.Blur()

		return model, nil

	case "enter":
		return model.selectCurrentCommand()

	case "up":
		model.moveCursor(-1)

		return model, nil

	case "down":
		model.moveCursor(1)

		return model, nil
	}

	previousValue := model.filterInput.Value()

	var command tea.Cmd
	model.filterInput, command = model.filterInput.Update(message)

	if model.filterInput.Value() != previousValue {
		model.applyFilter()
		model.cursor = 0
		model.offset = 0
		model.page = 0
		model.clampCursor()
		model.ensureCursorVisible()
	}

	return model, command
}

func (model *Model) moveCursor(delta int) {
	if len(model.filtered) == 0 {
		model.cursor = 0
		model.offset = 0
		model.page = 0
		return
	}

	model.cursor += delta

	if model.cursor < 0 {
		model.cursor = 0
	}

	if model.cursor >= len(model.filtered) {
		model.cursor = len(model.filtered) - 1
	}

	if !model.hasFilter() {
		model.page = model.cursor / model.pageSize
	}

	model.ensureCursorVisible()
}

func (model *Model) previousPage() {
	if model.hasFilter() || model.page == 0 {
		return
	}

	model.page--
	model.cursor = model.page * model.pageSize
	model.offset = 0
}

func (model *Model) nextPage() {
	if model.hasFilter() || model.page >= model.totalPages()-1 {
		return
	}

	model.page++
	model.cursor = model.page * model.pageSize
	model.offset = 0
}

func (model Model) hasFilter() bool {
	return strings.TrimSpace(model.filterInput.Value()) != ""
}

func (model Model) totalPages() int {
	if len(model.filtered) == 0 {
		return 1
	}

	return (len(model.filtered) + model.pageSize - 1) /
		model.pageSize
}

func (model *Model) applyFilter() {
	query := strings.TrimSpace(
		strings.ToLower(model.filterInput.Value()),
	)

	if query == "" {
		model.filtered = model.commands
		return
	}

	queryTerms := strings.Fields(query)
	matches := make([]filteredCommand, 0)

	for _, command := range model.commands {
		score, matchesQuery := commandMatchScore(
			command,
			queryTerms,
		)

		if !matchesQuery {
			continue
		}

		matches = append(matches, filteredCommand{
			command: command,
			score:   score,
		})
	}

	sort.SliceStable(matches, func(left, right int) bool {
		return matches[left].score > matches[right].score
	})

	model.filtered = make([]catalog.Command, 0, len(matches))

	for _, match := range matches {
		model.filtered = append(
			model.filtered,
			match.command,
		)
	}
}

func commandMatchScore(
	command catalog.Command,
	queryTerms []string,
) (int, bool) {
	name := strings.ToLower(command.Name)
	title := strings.ToLower(command.Title)
	description := strings.ToLower(command.Description)
	run := strings.ToLower(command.Run)

	labels := make([]string, len(command.Labels))
	for index, label := range command.Labels {
		labels[index] = strings.ToLower(label)
	}

	totalScore := 0

	for _, term := range queryTerms {
		termScore := 0

		switch {
		case title == term:
			termScore = 100

		case strings.HasPrefix(title, term):
			termScore = 80

		case strings.Contains(title, term):
			termScore = 60

		case name == term:
			termScore = 55

		case strings.Contains(name, term):
			termScore = 45

		case containsExact(labels, term):
			termScore = 40

		case containsPartial(labels, term):
			termScore = 30

		case strings.Contains(description, term):
			termScore = 20

		case strings.Contains(run, term):
			termScore = 10
		}

		if termScore == 0 {
			return 0, false
		}

		totalScore += termScore
	}

	return totalScore, true
}

func containsExact(values []string, query string) bool {
	for _, value := range values {
		if value == query {
			return true
		}
	}

	return false
}

func containsPartial(values []string, query string) bool {
	for _, value := range values {
		if strings.Contains(value, query) {
			return true
		}
	}

	return false
}

func (model *Model) clampCursor() {
	if len(model.filtered) == 0 {
		model.cursor = 0
		return
	}

	if model.cursor < 0 {
		model.cursor = 0
	}

	if model.cursor >= len(model.filtered) {
		model.cursor = len(model.filtered) - 1
	}
}

func (model *Model) ensureCursorVisible() {
	model.clampCursor()

	if len(model.filtered) == 0 {
		model.offset = 0
		model.page = 0
		return
	}

	if !model.hasFilter() {
		model.page = model.cursor / model.pageSize
		model.offset = 0
		return
	}

	availableHeight := model.availableListHeight()

	if model.cursor < model.offset {
		model.offset = model.cursor
	}

	for model.visibleHeight(model.offset, model.cursor) >
		availableHeight {

		model.offset++
	}

	if model.offset > model.cursor {
		model.offset = model.cursor
	}
}

func (model Model) visibleHeight(
	start int,
	end int,
) int {
	if end < start {
		return 0
	}

	return end - start + 1
}

func (model Model) availableListHeight() int {
	height := model.height - 9
	if height < 5 {
		height = 5
	}

	return height
}

func (model Model) View() tea.View {
	if model.detailMode {
		return model.detailView()
	}

	var builder strings.Builder

	builder.WriteString("\n")
	builder.WriteString("  ")
	builder.WriteString(titleStyle.Render("cmdpeek"))
	builder.WriteString("\n")

	builder.WriteString("  ")
	builder.WriteString(countStyle.Render(model.countText()))
	builder.WriteString("\n")

	builder.WriteString("\n  ")
	builder.WriteString(model.renderSearch())
	builder.WriteString("\n")

	builder.WriteString("\n")
	builder.WriteString(model.renderCommands())
	builder.WriteString("\n")

	builder.WriteString(helpStyle.Render(model.helpText()))
	builder.WriteString("\n")

	return tea.NewView(builder.String())
}

func (model Model) renderCommands() string {
	if len(model.filtered) == 0 {
		return "  " + noResultsStyle.Render("No matching commands") + "\n"
	}

	visible, offset := model.visibleCommands()
	var builder strings.Builder
	maxVisible := model.availableListHeight()

	for index, command := range visible {
		if index >= maxVisible {
			break
		}

		absoluteIndex := offset + index
		builder.WriteString(
			model.renderCommandLine(command, absoluteIndex == model.cursor),
		)
	}

	return builder.String()
}

func (model Model) visibleCommands() ([]catalog.Command, int) {
	if len(model.filtered) == 0 {
		return nil, 0
	}

	if !model.hasFilter() {
		start := model.page * model.pageSize
		end := start + model.pageSize

		if end > len(model.filtered) {
			end = len(model.filtered)
		}

		return model.filtered[start:end], start
	}

	start := model.offset
	if start >= len(model.filtered) {
		start = len(model.filtered) - 1
	}

	return model.filtered[start:], start
}

func (model Model) renderCommandLine(
	command catalog.Command,
	selected bool,
) string {
	marker := "  "
	title := normalCommandStyle.Render(command.Title)

	if selected {
		marker = selectedMarkerStyle.Render("▌ ")
		title = selectedCommandStyle.Render(command.Title)
	}

	typeSummary := commandVariableTypeSummary(command)
	if typeSummary != "" {
		title += "  " + commandTypeSummaryStyle.Render(typeSummary)
	}

	return marker + title + "\n"
}

func commandVariableTypeSummary(command catalog.Command) string {
	if len(command.Variables) == 0 {
		return "no variables"
	}

	types := make([]string, 0, len(command.Variables))
	seen := make(map[string]struct{}, len(command.Variables))

	for _, variable := range command.Variables {
		if _, exists := seen[variable.Source.Type]; exists {
			continue
		}

		seen[variable.Source.Type] = struct{}{}
		types = append(types, variable.Source.Type)
	}

	return strings.Join(types, " · ")
}

func (model Model) detailView() tea.View {
	var builder strings.Builder
	command, ok := model.currentCommand()

	builder.WriteString("\n  ")
	builder.WriteString(titleStyle.Render("cmdpeek"))
	builder.WriteString("\n")

	if !ok {
		builder.WriteString("\n  ")
		builder.WriteString(noResultsStyle.Render("No command selected"))
		builder.WriteString("\n")
		return tea.NewView(builder.String())
	}

	builder.WriteString("  ")
	builder.WriteString(countStyle.Render("Command details"))
	builder.WriteString("\n\n")

	lines := model.detailLines(command)
	start := model.detailOffset
	end := start + model.detailViewportHeight()
	if end > len(lines) {
		end = len(lines)
	}

	for _, line := range lines[start:end] {
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	builder.WriteString("\n")
	builder.WriteString(helpStyle.Render(model.detailFooter(len(lines))))
	builder.WriteString("\n")

	return tea.NewView(builder.String())
}

func (model Model) detailLines(command catalog.Command) []string {
	width := model.width - 8
	if width < 30 {
		width = 30
	}

	lines := []string{
		"  " + selectedCommandStyle.Render(command.Title),
		"",
	}

	lines = appendDetailField(lines, "Description", command.Description, width, descriptionValueStyle)
	lines = appendDetailField(lines, "Name", command.Name, width, nameValueStyle)
	lines = appendDetailField(lines, "Shell", command.Shell, width, nameValueStyle)
	lines = appendDetailField(lines, "Labels", strings.Join(command.Labels, " · "), width, labelsValueStyle)

	lines = append(lines, "", "  "+variablesHeadingStyle.Render("Command"))
	for _, line := range wrapText(commandtemplatePreview(command.Run), width) {
		lines = append(lines, "    "+commandValueStyle.Render(line))
	}

	if len(command.Variables) > 0 {
		lines = append(lines, "", "  "+variablesHeadingStyle.Render("Variables"))

		for _, variable := range command.Variables {
			lines = append(
				lines,
				"    "+variableNameStyle.Render(variable.Name)+"  "+variableTypeStyle.Render(variable.Source.Type),
			)

			description := variable.Description
			if description == "" {
				description = variable.Prompt
			}

			for _, line := range wrapText(description, width-4) {
				lines = append(lines, "      "+variableMetadataStyle.Render(line))
			}

			for _, line := range wrapText(variableSourceSummary(variable), width-4) {
				lines = append(lines, "      "+variableMetadataStyle.Render(line))
			}
		}
	}

	return lines
}

func appendDetailField(
	lines []string,
	label string,
	value string,
	width int,
	style lipgloss.Style,
) []string {
	if strings.TrimSpace(value) == "" {
		return lines
	}

	wrapped := wrapText(value, width-15)
	for index, line := range wrapped {
		prefix := "               "
		if index == 0 {
			prefix = "  " + detailLabelStyle.Render(label)
		}

		lines = append(lines, prefix+style.Render(line))
	}

	return lines
}

func wrapText(value string, width int) []string {
	if width < 10 {
		width = 10
	}

	var result []string
	for _, sourceLine := range strings.Split(value, "\n") {
		if sourceLine == "" {
			result = append(result, "")
			continue
		}

		remaining := []rune(sourceLine)
		for len(remaining) > width {
			breakAt := width
			for index := width; index > 0; index-- {
				if remaining[index-1] == ' ' || remaining[index-1] == '	' {
					breakAt = index - 1
					break
				}
			}

			if breakAt == 0 {
				breakAt = width
			}

			result = append(result, string(remaining[:breakAt]))
			remaining = remaining[breakAt:]
			for len(remaining) > 0 && (remaining[0] == ' ' || remaining[0] == '	') {
				remaining = remaining[1:]
			}
		}

		result = append(result, string(remaining))
	}

	return result
}

func (model Model) currentCommand() (catalog.Command, bool) {
	if len(model.filtered) == 0 || model.cursor >= len(model.filtered) {
		return catalog.Command{}, false
	}

	return model.filtered[model.cursor], true
}

func (model Model) detailViewportHeight() int {
	height := model.height - 8
	if height < 8 {
		height = 8
	}

	return height
}

func (model Model) maxDetailOffset() int {
	command, ok := model.currentCommand()
	if !ok {
		return 0
	}

	maximum := len(model.detailLines(command)) - model.detailViewportHeight()
	if maximum < 0 {
		return 0
	}

	return maximum
}

func (model *Model) clampDetailOffset() {
	maximum := model.maxDetailOffset()
	if model.detailOffset < 0 {
		model.detailOffset = 0
	}
	if model.detailOffset > maximum {
		model.detailOffset = maximum
	}
}

func (model Model) detailFooter(totalLines int) string {
	start := model.detailOffset + 1
	end := model.detailOffset + model.detailViewportHeight()
	if end > totalLines {
		end = totalLines
	}

	position := fmt.Sprintf("lines %d-%d of %d", start, end, totalLines)
	return "  " + position + "   ↑/↓ scroll   ←/→ page   enter select   esc back"
}

func variableSourceSummary(variable catalog.Variable) string {
	switch variable.Source.Type {
	case catalog.VariableSourceInput:
		if variable.Default != "" {
			return "default: " + variable.Default
		}
		return "manual input"

	case catalog.VariableSourceOptions:
		return "options: " + strings.Join(variable.Source.Values, " · ")

	case catalog.VariableSourceEnvironment:
		return "environment: " + variable.Source.Variable

	case catalog.VariableSourceCommand:
		uses := variablePattern.FindAllStringSubmatch(variable.Source.Command, -1)
		parts := make([]string, 0, 2)

		if len(uses) > 0 {
			names := make([]string, 0, len(uses))
			seen := make(map[string]struct{}, len(uses))

			for _, match := range uses {
				name := match[1]
				if _, exists := seen[name]; exists {
					continue
				}
				seen[name] = struct{}{}
				names = append(names, name)
			}

			parts = append(parts, "uses: "+strings.Join(names, ", "))
		}

		parts = append(parts, "source: "+commandtemplatePreview(variable.Source.Command))
		return strings.Join(parts, " · ")
	}

	return variable.Source.Type
}

func commandtemplatePreview(command string) string {
	return variablePattern.ReplaceAllString(command, "<$1>")
}

func renderDetailRow(label string, value string) string {
	return "    " +
		detailLabelStyle.Render(label) +
		value +
		"\n"
}

func renderCommandPreview(command string) string {
	matches := variablePattern.FindAllStringSubmatchIndex(
		command,
		-1,
	)

	if len(matches) == 0 {
		return commandValueStyle.Render(command)
	}

	var builder strings.Builder
	currentIndex := 0

	for _, match := range matches {
		builder.WriteString(
			commandValueStyle.Render(
				command[currentIndex:match[0]],
			),
		)

		variableName := command[match[2]:match[3]]

		builder.WriteString(
			variableStyle.Render(
				"<" + variableName + ">",
			),
		)

		currentIndex = match[1]
	}

	builder.WriteString(
		commandValueStyle.Render(command[currentIndex:]),
	)

	return builder.String()
}

func (model Model) countText() string {
	visible := len(model.filtered)
	total := len(model.commands)

	if model.hasFilter() {
		return fmt.Sprintf(
			"%d of %d commands",
			visible,
			total,
		)
	}

	return fmt.Sprintf(
		"Page %d/%d · %d commands",
		model.page+1,
		model.totalPages(),
		total,
	)
}

func (model Model) helpText() string {
	if model.filtering {
		return "  type to filter   ↑/↓ navigate   e details   enter select   esc close search"
	}

	if model.filterInput.Value() != "" {
		return "  / edit search   ↑/↓ navigate   e details   enter select   esc clear   q quit"
	}

	return "  / search   ↑/↓ navigate   ←/→ page   e details   enter select   q quit"
}

func Run(
	commands []catalog.Command,
) (*catalog.Command, error) {
	program := tea.NewProgram(New(commands))

	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("run TUI: %w", err)
	}

	model, ok := finalModel.(Model)
	if !ok {
		return nil, fmt.Errorf("unexpected final model type")
	}

	if !model.hasSelected {
		return nil, nil
	}

	selected := model.selected

	return &selected, nil
}

func (model Model) selectCurrentCommand() (tea.Model, tea.Cmd) {
	if len(model.filtered) == 0 {
		return model, nil
	}

	model.selected = model.filtered[model.cursor]
	model.hasSelected = true

	return model, tea.Quit
}

func (model Model) renderSearch() string {
	if model.filtering {
		return model.filterInput.View()
	}

	value := model.filterInput.Value()

	if value == "" {
		return countStyle.Render(
			"Search: press / to filter commands",
		)
	}

	return countStyle.Render(
		"Search: " + value,
	)
}
