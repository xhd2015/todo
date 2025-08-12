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
       todo <cmd> [OPTIONS]

Available sub commands:
  list
  export <file.json>
  import <file.json>
  config

Options:
  --storage <type>                 storage backend: sqlite (default), file, or server
  --server-addr <addr>             server address (required when --storage=server)
  --server-token <token>           server authentication token (optional when --storage=server)
  --debug-log <file>               enable debug logging to specified file
  -h,--help                        show this help message

Examples:
  todo                             run with SQLite storage (default)
  todo --storage=file              run with file storage (todos.json)
  todo --storage=server --server-addr=http://localhost:8080  run with server storage
  todo --storage=server --server-addr=http://localhost:8080 --server-token=abc123  run with server storage and auth
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
		case "config":
			return handleConfig(args[1:])
		}
	}

	var debugLogFile string
	var storageType string
	var serverAddr string
	var serverToken string

	var showPath bool

	args, err := flags.String("--storage", &storageType).
		String("--server-addr", &serverAddr).
		String("--server-token", &serverToken).
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

	// Apply config defaults
	storageConfig, err := ApplyConfigDefaults(storageType, serverAddr, serverToken)
	if err != nil {
		return err
	}
	storageType = storageConfig.StorageType
	serverAddr = storageConfig.ServerAddr
	serverToken = storageConfig.ServerToken

	// Validate server-addr is provided when storage type is server
	if storageType == "server" && serverAddr == "" {
		return fmt.Errorf("--server-addr is required when --storage=server")
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

	logManager, err := CreateLogManager(storageType, serverAddr, serverToken)
	if err != nil {
		return err
	}

	// Load config again to handle running PID (separate from storage config)
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
		Entries: logManager.Entries,
		Input: models.InputState{
			Focused: true,
		},
		SliceStart: -1,
		Refresh: func() {
			p.Send(cursor.Blink())
		},
		StatusBar: app.StatusBar{
			Storage: storageType,
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
	appState.OnAddChild = func(parentID int64, text string) (int64, error) {
		text = strings.TrimSpace(text)
		if text == "" {
			return 0, nil
		}
		id, err := logManager.Add(models.LogEntry{
			Text:     text,
			ParentID: parentID,
		})
		appState.Entries = logManager.Entries
		return id, err
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
		var foundEntry *models.LogEntryView
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
	appState.OnMove = func(id int64, newParentID int64) {
		logManager.Move(id, newParentID)
		appState.Entries = logManager.Entries
	}
	appState.OnAddNote = func(id int64, text string) {
		logManager.AddNote(id, models.Note{
			Text: text,
		})
		appState.Entries = logManager.Entries
	}
	appState.OnUpdateNote = func(entryID int64, noteID int64, text string) {
		logManager.UpdateNote(entryID, noteID, models.NoteOptional{
			Text: &text,
		})
		appState.Entries = logManager.Entries
	}
	appState.OnDeleteNote = func(entryID int64, noteID int64) {
		logManager.DeleteNote(entryID, noteID)
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
	m.app.Update(msg)
	if m.quit {
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) View() string {
	return m.app.Render()
}
