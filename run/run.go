package run

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xhd2015/go-dom-tui/charm"
	"github.com/xhd2015/go-dom-tui/log"
	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/todo/app"
	"github.com/xhd2015/todo/data"
	"github.com/xhd2015/todo/internal/config"
	"github.com/xhd2015/todo/internal/process"
	"github.com/xhd2015/todo/models"
)

const help = `
todo - A terminal-based todo list application

Usage: todo [OPTIONS]

Options:
  --storage <type>                 storage backend: sqlite (default) or file
  --debug-log <file>               enable debug logging to specified file
  -h,--help                        show this help message

Examples:
  todo                             run with SQLite storage (default)
  todo --storage=file              run with file storage (todos.json)
  todo --debug-log debug.log       run with debug logging enabled
`

func Main(args []string) error {
	if len(args) > 0 {
		arg0 := args[0]
		switch arg0 {
		case "list":
			return handleList(args[1:])
		case "export":
			return handleExport(args[1:])
		case "import":
			return handleImport(args[1:])
		}
	}

	var debugLogFile string
	var storageType string = "sqlite" // default to sqlite

	var showPath bool

	args, err := flags.String("--storage", &storageType).
		String("--debug-log", &debugLogFile).
		Bool("--show-path", &showPath).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unrecognized extra arguments: %s", strings.Join(args, " "))
	}

	confDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}

	if showPath {
		fmt.Println(confDir)
		return nil
	}

	err = os.MkdirAll(confDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	logManager, err := CreateLogManager(storageType)
	if err != nil {
		return err
	}
	config, err := data.LoadConfig()
	if err != nil {
		return err
	}

	if config != nil && config.RunningPID > 0 {
		exists, _ := process.ProcessExists(config.RunningPID)
		if exists {
			return fmt.Errorf("todo is already running with PID %d", config.RunningPID)
		}
	}
	if config == nil {
		config = &models.Config{}
	}
	config.RunningPID = os.Getpid()
	err = data.SaveConfig(config)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var openedFile *os.File
	if debugLogFile != "" {
		file, err := os.OpenFile(debugLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open debug log file: %w", err)
		}
		openedFile = file
		log.SetLogger(log.NewFileLogger(file))
	}

	err = logManager.Init()
	if err != nil {
		return err
	}

	var p *tea.Program
	appState := app.State{
		Entries:            logManager.Entries,
		SelectedEntryIndex: -1,
		EnteredEntryIndex:  -1,
		Input: models.InputState{
			Focused: true,
		},
		Refresh: func() {
			p.Send(cursor.Blink())
		},
	}
	appState.OnRefreshEntries = func() {
		// Run refresh asynchronously to avoid blocking the UI
		go func() {
			err := logManager.InitWithHistory(appState.ShowHistory)
			if err != nil {
				// TODO: Handle error appropriately
				return
			}
			appState.Entries = logManager.Entries
			p.Send(cursor.Blink()) // Trigger UI refresh
		}()
	}
	appState.OnAdd = func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		logManager.Add(models.LogEntry{
			Text: value,
		})
		appState.Entries = logManager.Entries
	}
	appState.OnAddChild = func(parentID int64, text string) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		logManager.Add(models.LogEntry{
			Text:     text,
			ParentID: parentID,
		})
		appState.Entries = logManager.Entries
	}
	appState.OnUpdate = func(id int64, text string) {
		logManager.Update(id, models.LogEntryOptional{
			Text: &text,
		})
		appState.Entries = logManager.Entries
	}
	appState.OnDelete = func(id int64) {
		logManager.Delete(id)
		appState.Entries = logManager.Entries
	}
	appState.OnToggle = func(id int64) {
		var foundEntry *models.EntryView
		for _, entry := range logManager.Entries {
			if entry.Data.ID == id {
				foundEntry = entry
				break
			}
		}
		if foundEntry == nil {
			return
		}
		done := !foundEntry.Data.Done
		var doneTime *time.Time
		if done {
			now := time.Now()
			doneTime = &now
		}
		logManager.Update(id, models.LogEntryOptional{
			Done:     &done,
			DoneTime: &doneTime,
		})
		appState.Entries = logManager.Entries
	}
	appState.OnPromote = func(id int64) {
		currentTime := time.Now().UnixMilli()
		logManager.Update(id, models.LogEntryOptional{
			AdjustedTopTime: &currentTime,
		})
		appState.Entries = logManager.Entries
	}
	appState.OnUpdateHighlight = func(id int64, highlightLevel int) {
		logManager.Update(id, models.LogEntryOptional{
			HighlightLevel: &highlightLevel,
		})
		appState.Entries = logManager.Entries
	}
	appState.OnAddNote = func(id int64, text string) {
		logManager.AddNote(id, models.Note{
			Text: text,
		})
		appState.Entries = logManager.Entries
	}

	model := &Model{
		app: charm.NewCharmApp(&appState, app.App),
	}

	appState.Quit = func() {
		model.quit = true
		if openedFile != nil {
			openedFile.Close()
		}
	}

	p = tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

type Model struct {
	quit bool
	app  *charm.CharmApp[app.State]
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	res := m.app.Update(msg)
	if m.quit {
		return m, tea.Quit
	}
	if res, ok := res.(tea.Cmd); ok {
		return m, res
	}
	return m, nil
}

func (m *Model) View() string {
	return m.app.Render()
}
