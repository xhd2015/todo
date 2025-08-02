package run

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/process"
)

// Configurable constants
const (
	CtrlCExitDelayMs = 1000 // Maximum delay between two Ctrl+C presses to exit
)

func Main(args []string) error {
	m, err := initialModel()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

type Todo struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type Config struct {
	LastInput  string `json:"last_input"`
	RunningPID int    `json:"running_pid"`
}

type model struct {
	todos            []Todo
	cursor           int
	input            textinput.Model
	quitting         bool
	configDir        string
	windowHeight     int
	windowWidth      int
	lastCtrlC        time.Time
	showExitPrompt   bool
	focusOnInput     bool
	showConfirm      bool
	confirmText      string
	confirmButton    int  // 0 = OK, 1 = Cancel
	isInfoDialog     bool // true if this is just showing info, false if asking to add todo
	editingTodo      int  // -1 = not editing, otherwise index of todo being edited
	editInput        textinput.Model
	editButton       int  // 0 = Save, 1 = Cancel
	editFocusOnInput bool // true = focus on edit input, false = focus on buttons
}

var (
	checkedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	uncheckedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	strikeStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Strikethrough(true)
	cursorStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	inputStyle          = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	warningStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	confirmStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	selectedButtonStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3")).Bold(true)
)

func initialModel() (model, error) {
	ti := textinput.New()
	ti.Placeholder = "/add <description> to add, /exit or q to quit, space to toggle"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 80

	editTi := textinput.New()
	editTi.CharLimit = 256
	editTi.Width = 80

	configDir, err := os.UserConfigDir()
	if err != nil {
		return model{}, err
	}
	configDir = filepath.Join(configDir, "go-todo")

	m := model{
		input:        ti,
		configDir:    configDir,
		focusOnInput: true,
		editingTodo:  -1,
		editInput:    editTi,
	}

	// Check if another instance is already running
	err = m.checkRunningInstance()
	if err != nil {
		return model{}, err
	}

	err = m.loadTodos()
	if err != nil {
		return m, err
	}

	err = m.loadConfig()
	if err != nil {
		return m, err
	}

	// Record current PID as running
	err = m.saveConfig()
	if err != nil {
		return m, err
	}

	return m, nil
}

func (m *model) loadConfig() error {
	configPath := filepath.Join(m.configDir, "config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // No config file yet, that's ok
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Restore last input
	m.input.SetValue(config.LastInput)
	return nil
}

func (m *model) saveConfig() error {
	configPath := filepath.Join(m.configDir, "config.json")

	// Don't save control commands that have been consumed
	inputToSave := m.input.Value()
	if isControlCommand(inputToSave) {
		inputToSave = "" // Clear control commands
	}

	config := Config{
		LastInput:  inputToSave,
		RunningPID: os.Getpid(),
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func (m *model) clearPID() error {
	configPath := filepath.Join(m.configDir, "config.json")

	// Don't save control commands that have been consumed
	inputToSave := m.input.Value()
	if isControlCommand(inputToSave) {
		inputToSave = "" // Clear control commands
	}

	config := Config{
		LastInput:  inputToSave,
		RunningPID: 0, // Clear PID
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func isControlCommand(input string) bool {
	trimmed := strings.TrimSpace(input)
	controlCommands := []string{
		"/exit", "exit", "quit", "/q", "/quit", "/config",
	}

	for _, cmd := range controlCommands {
		if trimmed == cmd {
			return true
		}
	}

	// Also check if it starts with /add (these are processed, not restored)
	return strings.HasPrefix(trimmed, "/add ")
}

func processExists(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}

	// Check if process exists by sending signal 0
	_, findErr := os.FindProcess(pid)
	if findErr != nil {
		return false, nil
	}

	return isProcessAlive(pid)
}

func isProcessAlive(pid int) (bool, error) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return false, nil
		}
		return false, fmt.Errorf("failed to find process: %v", err)
	}

	// Check if the process is running
	isRunning, err := p.IsRunning()
	if err != nil {
		return false, fmt.Errorf("failed to check if process is running: %v", err)
	}

	return isRunning, nil
}

func (m *model) checkRunningInstance() error {
	configPath := filepath.Join(m.configDir, "config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // No config file yet, safe to proceed
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil // Can't read config, assume safe to proceed
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil // Invalid config, assume safe to proceed
	}

	if config.RunningPID > 0 {
		exists, _ := processExists(config.RunningPID)
		if exists {
			return fmt.Errorf("todo is already running with PID %d", config.RunningPID)
		}
	}

	return nil
}

func (m *model) loadTodos() error {
	todoPath := filepath.Join(m.configDir, "todos.json")

	if _, err := os.Stat(todoPath); os.IsNotExist(err) {
		err = os.MkdirAll(m.configDir, 0755)
		if err != nil {
			return err
		}
		m.todos = []Todo{}
		return m.saveTodos()
	}

	data, err := os.ReadFile(todoPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &m.todos)
}

func (m *model) saveTodos() error {
	todoPath := filepath.Join(m.configDir, "todos.json")
	data, err := json.MarshalIndent(m.todos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(todoPath, data, 0644)
}

func (m *model) addTodo(text string) {
	nextID := 1
	for _, todo := range m.todos {
		if todo.ID >= nextID {
			nextID = todo.ID + 1
		}
	}

	m.todos = append(m.todos, Todo{
		ID:   nextID,
		Text: text,
		Done: false,
	})
	m.saveTodos()
}

func (m *model) toggleTodo() {
	if len(m.todos) > 0 && m.cursor >= 0 && m.cursor < len(m.todos) {
		m.todos[m.cursor].Done = !m.todos[m.cursor].Done
		m.saveTodos()
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			now := time.Now()
			if !m.lastCtrlC.IsZero() && now.Sub(m.lastCtrlC) < time.Duration(CtrlCExitDelayMs)*time.Millisecond {
				// Double Ctrl+C within delay period - save config and exit
				m.clearPID() // Clear PID and save current input state
				m.quitting = true
				return m, tea.Quit
			} else {
				// First Ctrl+C or too long since last one - show prompt
				m.lastCtrlC = now
				m.showExitPrompt = true
				return m, tea.Tick(time.Duration(CtrlCExitDelayMs)*time.Millisecond, func(t time.Time) tea.Msg {
					return "clear_exit_prompt"
				})
			}

		case "q":
			m.clearPID() // Clear PID and save current input state
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			m.showExitPrompt = false
			if m.showConfirm {
				// Navigate between OK/Cancel buttons (only OK for info dialogs)
				if !m.isInfoDialog {
					m.confirmButton = 0 // OK
				}
			} else if m.editingTodo >= 0 {
				// In edit mode - move focus from buttons to input
				if !m.editFocusOnInput {
					m.editFocusOnInput = true
					m.editInput.Focus()
				}
			} else if m.focusOnInput {
				// Move focus from input to todo list if at beginning of input
				if m.input.Position() == 0 && len(m.todos) > 0 {
					m.focusOnInput = false
					m.input.Blur()
					m.cursor = len(m.todos) - 1 // Start at bottom of todo list
				}
			} else {
				// Navigate within todo list
				if m.cursor > 0 {
					m.cursor--
				}
			}

		case "down", "j":
			m.showExitPrompt = false
			if m.showConfirm {
				// Navigate between OK/Cancel buttons (only OK for info dialogs)
				if !m.isInfoDialog {
					m.confirmButton = 1 // Cancel
				}
			} else if m.editingTodo >= 0 {
				// In edit mode - move focus from input to buttons
				if m.editFocusOnInput {
					m.editFocusOnInput = false
					m.editInput.Blur()
				}
			} else if !m.focusOnInput {
				// Navigate within todo list
				if m.cursor < len(m.todos)-1 {
					m.cursor++
				} else {
					// At bottom of todo list, move focus back to input
					m.focusOnInput = true
					m.input.Focus()
				}
			}

		case "left", "h":
			if m.showConfirm {
				if !m.isInfoDialog {
					m.confirmButton = 0 // OK
				}
			} else if m.editingTodo >= 0 && !m.editFocusOnInput {
				// Only change buttons when focus is on buttons, not input
				m.editButton = 0 // Save
			}

		case "right", "l":
			if m.showConfirm {
				if !m.isInfoDialog {
					m.confirmButton = 1 // Cancel
				}
			} else if m.editingTodo >= 0 && !m.editFocusOnInput {
				// Only change buttons when focus is on buttons, not input
				m.editButton = 1 // Cancel
			}

		case "e":
			// Edit todo item if not focused on input
			if !m.focusOnInput && !m.showConfirm && m.editingTodo < 0 && len(m.todos) > 0 && m.cursor >= 0 && m.cursor < len(m.todos) {
				m.editingTodo = m.cursor
				m.editInput.SetValue(m.todos[m.cursor].Text)
				m.editInput.Focus()
				m.editButton = 0          // Default to Save
				m.editFocusOnInput = true // Start with focus on input
				return m, nil             // Return early to prevent 'e' from being processed by input
			}

		case " ":
			m.showExitPrompt = false
			if m.showConfirm {
				// Space acts like Enter in confirm dialog
				if !m.isInfoDialog && m.confirmButton == 0 { // Only add todo if not info dialog and OK selected
					m.addTodo(m.confirmText)
					m.input.SetValue("")
				}
				// Always close the dialog
				m.showConfirm = false
				m.confirmText = ""
				m.confirmButton = 0
				m.isInfoDialog = false
			} else if m.editingTodo >= 0 {
				// Space acts like Enter in edit mode (only when on buttons)
				if !m.editFocusOnInput {
					if m.editButton == 0 { // Save
						m.todos[m.editingTodo].Text = m.editInput.Value()
						m.saveTodos()
					}
					// Save or Cancel both close the edit mode
					m.editingTodo = -1
					m.editInput.Blur()
					m.editButton = 0
					m.editFocusOnInput = false
				} else {
					// When focus is on input, let input handle space
					m.editInput, cmd = m.editInput.Update(msg)
					return m, cmd
				}
			} else if !m.focusOnInput {
				// Toggle todo when focus is on todo list
				m.toggleTodo()
			} else {
				// Let input handle space when focused
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}

		case "enter":
			m.showExitPrompt = false
			if m.showConfirm {
				// Handle confirmation dialog
				if !m.isInfoDialog && m.confirmButton == 0 { // Only add todo if not info dialog and OK selected
					m.addTodo(m.confirmText)
					m.input.SetValue("")
				} else if !m.isInfoDialog && m.confirmButton == 1 { // Cancel - restore text
					m.input.SetValue(m.confirmText)
				}
				// Always close the dialog
				m.showConfirm = false
				m.confirmText = ""
				m.confirmButton = 0
				m.isInfoDialog = false
			} else if m.editingTodo >= 0 {
				// Handle edit mode
				if m.editButton == 0 { // Save
					m.todos[m.editingTodo].Text = m.editInput.Value()
					m.saveTodos()
				}
				// Save or Cancel both close the edit mode
				m.editingTodo = -1
				m.editInput.Blur()
				m.editButton = 0
				m.editFocusOnInput = false
			} else if !m.focusOnInput {
				// Enter in todo list moves focus back to input
				m.focusOnInput = true
				m.input.Focus()
			} else {
				// Handle commands when input is focused
				value := m.input.Value()
				if strings.HasPrefix(value, "/add ") {
					todoText := strings.TrimSpace(value[5:])
					if todoText != "" {
						m.addTodo(todoText)
						m.input.SetValue("")
					}
				} else if value == "/exit" || value == "exit" || value == "quit" || value == "/q" || value == "/quit" {
					m.clearPID() // Clear PID and save current input state
					m.quitting = true
					return m, tea.Quit
				} else if value == "/config" {
					// Show config directory path (don't clear input)
					m.showConfirm = true
					m.confirmText = fmt.Sprintf("Config dir: %s", m.configDir)
					m.confirmButton = 0   // Default to OK
					m.isInfoDialog = true // This is just showing info
					m.input.SetValue("")  // Clear the /config command
				} else if strings.TrimSpace(value) != "" {
					// Show confirmation dialog for non-command text
					m.showConfirm = true
					m.confirmText = strings.TrimSpace(value)
					m.confirmButton = 0    // Default to OK
					m.isInfoDialog = false // This is asking to add todo
				}
			}
		}

	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.windowWidth = msg.Width
		m.input.Width = msg.Width - 4
		m.editInput.Width = msg.Width - 4

	case string:
		if msg == "clear_exit_prompt" {
			m.showExitPrompt = false
		}
	}

	// Update appropriate input based on state
	if m.focusOnInput {
		m.input, cmd = m.input.Update(msg)
	} else if m.editingTodo >= 0 && m.editFocusOnInput {
		m.editInput, cmd = m.editInput.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var todoContent strings.Builder
	todoContent.WriteString("Todo List\n\n")

	for i, todo := range m.todos {
		cursor := " "
		if !m.focusOnInput && m.cursor == i && m.editingTodo < 0 {
			cursor = cursorStyle.Render(">")
		} else {
			cursor = " "
		}

		if m.editingTodo == i {
			// Show edit mode with input and buttons
			editBox := inputStyle.Render(m.editInput.View())

			saveButton := "[Save]"
			cancelButton := "[Cancel]"

			if m.editButton == 0 {
				saveButton = selectedButtonStyle.Render("[Save]")
			} else {
				cancelButton = selectedButtonStyle.Render("[Cancel]")
			}

			editSection := saveButton + " " + cancelButton

			// Calculate spacing for alignment
			editBoxWidth := lipgloss.Width(editBox)
			editSectionWidth := lipgloss.Width(editSection)
			totalAvailableWidth := m.windowWidth - 4 // Account for margins

			if totalAvailableWidth > editBoxWidth+editSectionWidth+2 {
				spacing := totalAvailableWidth - editBoxWidth - editSectionWidth
				todoContent.WriteString(fmt.Sprintf("%s   %s%s%s\n", cursor, editBox, strings.Repeat(" ", spacing), editSection))
			} else {
				// If not enough space, put buttons under the edit box
				todoContent.WriteString(fmt.Sprintf("%s   %s\n", cursor, editBox))
				todoContent.WriteString(fmt.Sprintf("    %s\n", editSection))
			}
		} else {
			// Show normal todo item
			checkbox := "☐"
			text := todo.Text
			if todo.Done {
				checkbox = checkedStyle.Render("☑")
				text = strikeStyle.Render(text)
			} else {
				checkbox = uncheckedStyle.Render(checkbox)
			}

			todoContent.WriteString(fmt.Sprintf("%s %s %s\n", cursor, checkbox, text))
		}
	}

	if len(m.todos) == 0 {
		todoContent.WriteString("  No todos yet. Use /add <description> to add one!\n")
	}

	// Calculate available space for todo content
	inputHeight := 4 // Input box + borders + controls text
	availableHeight := m.windowHeight - inputHeight
	if availableHeight < 3 {
		availableHeight = 3
	}

	todoLines := strings.Split(todoContent.String(), "\n")
	todoContentStr := todoContent.String()

	// If content is too long, we'll still show it all and let it scroll
	// This ensures input stays at bottom

	// Create the bottom section (input + controls)
	inputBox := inputStyle.Render(m.input.View())

	var inputLine string
	if m.showConfirm {
		// Create confirmation prompt on same line, left-aligned
		var confirmMsg string
		var okButton, cancelButton string
		var confirmSection string

		if m.isInfoDialog {
			// For info dialogs, just show the message with OK button
			confirmMsg = confirmStyle.Render(m.confirmText + " ")
			okButton = "[OK]"

			if m.confirmButton == 0 {
				okButton = selectedButtonStyle.Render("[OK]")
			}

			confirmSection = confirmMsg + okButton
		} else {
			// For add todo dialogs, show "Add to list?" with OK/Cancel
			confirmMsg = confirmStyle.Render("Add to list? ")
			okButton = "[OK]"
			cancelButton = "[Cancel]"

			if m.confirmButton == 0 {
				okButton = selectedButtonStyle.Render("[OK]")
			} else {
				cancelButton = selectedButtonStyle.Render("[Cancel]")
			}

			confirmSection = confirmMsg + okButton + " " + cancelButton
		}

		// Put confirmation on left, input on same line if space allows
		confirmWidth := lipgloss.Width(confirmSection)
		inputWidth := lipgloss.Width(inputBox)
		totalAvailableWidth := m.windowWidth - 4 // Account for margins

		if totalAvailableWidth > inputWidth+confirmWidth+2 {
			spacing := totalAvailableWidth - inputWidth - confirmWidth
			inputLine = "\n" + confirmSection + strings.Repeat(" ", spacing) + inputBox
		} else {
			// If not enough space, put confirmation under the input
			inputLine = "\n" + inputBox + "\n" + confirmSection
		}
	} else {
		inputLine = "\n" + inputBox
	}

	controlsText := "\nControls: ↑/↓ to navigate, space to toggle, /add <text> to add, q to quit"

	// Show exit prompt if Ctrl+C was pressed once
	if m.showExitPrompt {
		controlsText = "\n" + warningStyle.Render("Press Ctrl+C again to exit") + controlsText
	}

	inputSection := inputLine + controlsText

	// Calculate padding to push input to bottom
	contentHeight := len(todoLines)
	if contentHeight < availableHeight-3 { // Reserve space for controls
		padding := availableHeight - contentHeight - 3
		if padding > 0 {
			todoContentStr += strings.Repeat("\n", padding)
		}
	}

	return todoContentStr + inputSection
}
