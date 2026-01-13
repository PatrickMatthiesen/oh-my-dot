package interactive

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

type Mode int

const (
	ModeAuto           Mode = iota // Smart detection
	ModeInteractive                // Force interactive
	ModeNonInteractive             // Force non-interactive
)

// GetMode detects the appropriate mode based on flags and environment
func GetMode(cmd *cobra.Command) Mode {
	// 1. Explicit flags take precedence
	if interactive, _ := cmd.Flags().GetBool("interactive"); interactive {
		return ModeInteractive
	}

	if noInteractive, _ := cmd.Flags().GetBool("no-interactive"); noInteractive {
		return ModeNonInteractive
	}

	// 2. Check environment variables (for CI)
	if os.Getenv("CI") != "" ||
		os.Getenv("OMDOT_NON_INTERACTIVE") != "" {
		return ModeNonInteractive
	}

	// 3. Check if running in a TTY
	if !isatty.IsTerminal(os.Stdin.Fd()) ||
		!isatty.IsTerminal(os.Stdout.Fd()) {
		return ModeNonInteractive
	}

	// 4. Default: Auto mode (smart behavior)
	return ModeAuto
}

// ShouldPrompt determines if we should prompt for this specific scenario
func ShouldPrompt(cmd *cobra.Command, hasRequiredInfo bool) bool {
	mode := GetMode(cmd)

	switch mode {
	case ModeInteractive:
		return true
	case ModeNonInteractive:
		return false
	case ModeAuto:
		// Only prompt if we're missing required information
		return !hasRequiredInfo
	}

	return false
}

// PromptInput prompts the user for text input
func PromptInput(question string, defaultValue string) (string, error) {
	m := textInputModel{
		textInput: textinput.New(),
		question:  question,
	}
	m.textInput.Placeholder = defaultValue
	m.textInput.Focus()
	m.textInput.CharLimit = 256
	m.textInput.SetWidth(50)

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	if finalModel, ok := result.(textInputModel); ok {
		if finalModel.cancelled {
			return "", fmt.Errorf("cancelled")
		}
		value := finalModel.textInput.Value()
		if value == "" {
			return defaultValue, nil
		}
		return value, nil
	}

	return "", fmt.Errorf("unexpected model type")
}

type textInputModel struct {
	textInput textinput.Model
	question  string
	cancelled bool
}

func (m textInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.Key()
		switch key.Code {
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeyEscape:
			m.cancelled = true
			return m, tea.Quit
		}
		// Check for ctrl+c
		if msg.String() == "ctrl+c" {
			m.cancelled = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m textInputModel) View() tea.View {
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		lipgloss.NewStyle().Bold(true).Render(m.question),
		m.textInput.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(Press Enter to confirm, Esc to cancel)"),
	)
	return tea.NewView(content)
}

// PromptConfirm prompts the user for yes/no confirmation
func PromptConfirm(question string) (bool, error) {
	m := confirmModel{
		question: question,
		selected: false,
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return false, err
	}

	if finalModel, ok := result.(confirmModel); ok {
		if finalModel.cancelled {
			return false, fmt.Errorf("cancelled")
		}
		return finalModel.selected, nil
	}

	return false, fmt.Errorf("unexpected model type")
}

type confirmModel struct {
	question  string
	selected  bool
	cancelled bool
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.selected = true
			return m, tea.Quit
		case "n", "N":
			m.selected = false
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "left", "h":
			m.selected = false
		case "right", "l":
			m.selected = true
		}
	}
	return m, nil
}

func (m confirmModel) View() tea.View {
	yesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	noStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	if m.selected {
		yesStyle = yesStyle.Bold(true).Foreground(lipgloss.Color("42"))
	} else {
		noStyle = noStyle.Bold(true).Foreground(lipgloss.Color("196"))
	}

	content := fmt.Sprintf(
		"%s\n\n  %s  %s\n\n%s",
		lipgloss.NewStyle().Bold(true).Render(m.question),
		yesStyle.Render("[Yes]"),
		noStyle.Render("[No]"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(Use arrow keys or y/n, Enter to confirm, Esc to cancel)"),
	)
	return tea.NewView(content)
}

// PromptSelect prompts the user to select one item from a list
func PromptSelect(question string, options []string) (int, error) {
	items := make([]list.Item, len(options))
	for i, opt := range options {
		items[i] = item{title: opt, index: i}
	}

	const defaultWidth = 50
	const defaultHeight = 14

	l := list.New(items, itemDelegate{}, defaultWidth, defaultHeight)
	l.Title = question
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Padding(0, 1)

	m := selectModel{
		list: l,
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return -1, err
	}

	if finalModel, ok := result.(selectModel); ok {
		if finalModel.cancelled {
			return -1, fmt.Errorf("cancelled")
		}
		if i, ok := finalModel.list.SelectedItem().(item); ok {
			return i.index, nil
		}
	}

	return -1, fmt.Errorf("no selection made")
}

type item struct {
	title string
	index int
}

func (i item) FilterValue() string { return i.title }
func (i item) Title() string       { return i.title }
func (i item) Description() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := i.Title()

	fn := lipgloss.NewStyle().PaddingLeft(2).Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Bold(true).
				PaddingLeft(2).
				Render("→ " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type selectModel struct {
	list      list.Model
	cancelled bool
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m selectModel) View() tea.View {
	return tea.NewView(m.list.View())
}

// PromptMultiSelect prompts the user to select multiple items from a list
func PromptMultiSelect(question string, options []string) ([]int, error) {
	m := multiSelectModel{
		question: question,
		options:  options,
		selected: make(map[int]bool),
		cursor:   0,
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	if finalModel, ok := result.(multiSelectModel); ok {
		if finalModel.cancelled {
			return nil, fmt.Errorf("cancelled")
		}

		selectedIndices := []int{}
		for i := range finalModel.options {
			if finalModel.selected[i] {
				selectedIndices = append(selectedIndices, i)
			}
		}
		return selectedIndices, nil
	}

	return nil, fmt.Errorf("unexpected model type")
}

type multiSelectModel struct {
	question  string
	options   []string
	selected  map[int]bool
	cursor    int
	cancelled bool
}

func (m multiSelectModel) Init() tea.Cmd {
	return nil
}

func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "space":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "enter":
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m multiSelectModel) View() tea.View {
	s := lipgloss.NewStyle().Bold(true).Render(m.question) + "\n\n"

	for i, opt := range m.options {
		cursor := " "
		if m.cursor == i {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Render("→")
		}

		checked := " "
		if m.selected[i] {
			checked = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
		}

		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, opt)
	}

	s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(Use arrows to move, Space to select, Enter to confirm, Esc to cancel)")

	return tea.NewView(s)
}

// MultiSelect is a convenience wrapper around PromptMultiSelect that returns selected values
func MultiSelect(question string, options []string, defaultSelected func(string) bool) ([]string, error) {
	// Mark default selections
	m := multiSelectModel{
		question: question,
		options:  options,
		selected: make(map[int]bool),
		cursor:   0,
	}

	// Apply default selections
	if defaultSelected != nil {
		for i, opt := range options {
			if defaultSelected(opt) {
				m.selected[i] = true
			}
		}
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	if finalModel, ok := result.(multiSelectModel); ok {
		if finalModel.cancelled {
			return nil, fmt.Errorf("cancelled")
		}

		selected := []string{}
		for i := range finalModel.options {
			if finalModel.selected[i] {
				selected = append(selected, finalModel.options[i])
			}
		}
		return selected, nil
	}

	return nil, fmt.Errorf("unexpected model type")
}

// Confirm is a convenience wrapper around PromptConfirm with default value support
func Confirm(question string, defaultYes bool) (bool, error) {
	m := confirmModel{
		question: question,
		selected: defaultYes,
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return false, err
	}

	if finalModel, ok := result.(confirmModel); ok {
		if finalModel.cancelled {
			return false, fmt.Errorf("cancelled")
		}
		return finalModel.selected, nil
	}

	return false, fmt.Errorf("unexpected model type")
}

// PromptFilePicker prompts the user with a file picker for multi-file selection
// Returns a slice of absolute file paths that were selected.
// Users must explicitly select files with Space key; Enter with no selection cancels.
func PromptFilePicker(prompt string, directory string) ([]string, error) {
	if directory == "" {
		var err error
		directory, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	fp := filepicker.New()
	fp.CurrentDirectory = directory
	// AllowedTypes is nil by default, allowing all file types
	fp.ShowHidden = false
	fp.FileAllowed = true
	fp.DirAllowed = false

	m := filePickerModel{
		filepicker: fp,
		prompt:     prompt,
		selected:   make(map[string]bool),
		keyMap:     defaultFilePickerKeyMap(),
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	if finalModel, ok := result.(filePickerModel); ok {
		if finalModel.cancelled {
			return nil, fmt.Errorf("cancelled")
		}

		// Convert map to slice
		files := make([]string, 0, len(finalModel.selected))
		for file := range finalModel.selected {
			files = append(files, file)
		}

		return files, nil
	}

	return nil, fmt.Errorf("unexpected model type")
}

type filePickerModel struct {
	filepicker filepicker.Model
	prompt     string
	selected   map[string]bool
	cancelled  bool
	err        error
	keyMap     filePickerKeyMap
}

type filePickerKeyMap struct {
	Cancel  key.Binding
	Toggle  key.Binding
	Confirm key.Binding
}

func defaultFilePickerKeyMap() filePickerKeyMap {
	return filePickerKeyMap{
		Cancel:  key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel")),
		Toggle:  key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle selection")),
		Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	}
}

func (m filePickerModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m filePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle our keys BEFORE passing to filepicker
		switch {
		case key.Matches(msg, m.keyMap.Cancel):
			m.cancelled = true
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.Toggle):
			// Toggle selection - only allow files, not directories
			selectedPath := m.filepicker.HighlightedPath()
			if selectedPath == "" {
				// No file currently highlighted
				return m, nil
			}

			// Check if it's a file (not a directory)
			fileInfo, err := os.Stat(selectedPath)
			if err != nil || fileInfo.IsDir() {
				// Don't select directories, only files
				return m, nil
			}

			if m.selected[selectedPath] {
				delete(m.selected, selectedPath)
			} else {
				m.selected[selectedPath] = true
			}
			return m, nil
		case key.Matches(msg, m.keyMap.Confirm):
			// If nothing selected, don't auto-select - let user explicitly select with space
			if len(m.selected) == 0 {
				// Do nothing - require explicit selection with space
				return m, nil
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		m.selected[path] = true
	}

	return m, cmd
}

func (m filePickerModel) View() tea.View {
	if m.err != nil {
		return tea.NewView("Error: " + m.err.Error())
	}

	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Render(m.prompt) + "\n\n")
	s.WriteString(m.filepicker.View() + "\n\n")

	if len(m.selected) > 0 {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(fmt.Sprintf("Selected %d file(s):", len(m.selected))) + "\n")

		// Sort files for consistent display order
		files := make([]string, 0, len(m.selected))
		for file := range m.selected {
			files = append(files, file)
		}
		// Sort by just the filename for cleaner display
		sort.Slice(files, func(i, j int) bool {
			return files[i] < files[j]
		})

		for _, file := range files {
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  • "+file) + "\n")
		}
		s.WriteString("\n")
	}

	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(Space to toggle, Enter to confirm, Esc to cancel)"))

	return tea.NewView(s.String())
}
