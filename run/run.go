package run

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xhd2015/go-dom-tui/charm"
	"github.com/xhd2015/go-dom-tui/log"
	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/todo/app"
	"github.com/xhd2015/todo/app/exp"
	"github.com/xhd2015/todo/app/human_state"
	"github.com/xhd2015/todo/data"
	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/internal/config"
	"github.com/xhd2015/todo/internal/macos"
	"github.com/xhd2015/todo/internal/process"
	applog "github.com/xhd2015/todo/log"
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/run/tool"
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
  tool

Options:
  --storage <type>                 storage backend: file (default), sqlite, or server
  --server-addr <addr>             server address (required when --storage=server)
  --server-token <token>           server authentication token (optional when --storage=server)
  --debug-log <file>               enable debug logging to specified file
  --show-path                      show config path
  -h,--help                        show this help message

Examples:
  todo                             run with file storage (default)
  todo --storage=sqlite            run with SQLite storage
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
		case "tool":
			return handleTool(args[1:])
		}
	}

	var debugLogFile string
	var storageType string
	var serverAddr string
	var serverToken string

	// into group mode
	var group bool
	var humanStat bool
	var learning bool

	var showPath bool

	args, err := flags.String("--storage", &storageType).
		String("--server-addr", &serverAddr).
		String("--server-token", &serverToken).
		String("--debug-log", &debugLogFile).
		Bool("--show-path", &showPath).
		Bool("--group", &group).
		Bool("--human-stat,--hstat", &humanStat).
		Bool("--learning", &learning).
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

	// Initialize logging
	if err := applog.Init(); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

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

	logManager, services, err := CreateLogManager(storageType, serverAddr, serverToken)
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
		GroupCollapseState: app.NewMutexMap(),
		Refresh: func() {
			p.Send(cursor.Blink())
		},
		StatusBar: app.StatusBar{
			Storage: storageType,
		},
	}

	// Initialize SubmitState with restore callback
	appState.SubmitState.SetOnRestore(appState.Input.Append)
	appState.ChildSubmitState.SetOnRestore(appState.ChildInputState.Append)

	refreshEntries := func() {
		err := logManager.InitWithHistory(appState.ShowHistory)
		if err != nil {
			// TODO: Handle error appropriately
			appState.StatusBar.Error = err.Error()
			return
		}
		appState.Entries = logManager.Entries
	}

	appState.RefreshEntries = func(ctx context.Context) error {
		// Run refresh asynchronously to avoid blocking the UI
		refreshEntries()
		return nil
	}
	appState.OnAdd = func(ctx context.Context, viewType models.LogEntryViewType, value string) error {
		if viewType != models.LogEntryViewType_Log {
			return nil
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}

		return appState.SubmitState.Do(ctx, value, func() error {
			_, err := logManager.Add(models.LogEntry{
				Text: value,
			})
			if err != nil {
				return err
			}
			// Success: update entries
			appState.Entries = logManager.Entries
			return nil
		})
	}
	appState.OnAddChild = func(ctx context.Context, viewType models.LogEntryViewType, parentID int64, text string) (models.LogEntryViewType, int64, error) {
		if viewType != models.LogEntryViewType_Log && viewType != models.LogEntryViewType_Group {
			return models.LogEntryViewType_Log, 0, nil
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return models.LogEntryViewType_Log, 0, nil
		}
		var id int64
		var subEntryType models.LogEntryViewType
		submit := func() error {
			// Check if the parent is a group entry
			if viewType == models.LogEntryViewType_Group {
				// Create a rootless log entry (ParentID = 0) and bind to group
				logID, err := logManager.Add(models.LogEntry{
					Text: text,
				})
				if err != nil {
					return err
				}

				// Bind the new log to the group
				_, err = exp.LogGroupStore.Add(&exp.LogIDGroupMapping{
					LogID:   logID,
					GroupID: parentID,
				})
				if err != nil {
					return fmt.Errorf("failed to bind log to group: %w", err)
				}

				appState.Entries = logManager.Entries
				id = logID
				subEntryType = models.LogEntryViewType_Log
				return nil
			}

			// Regular child entry creation for log entries
			var err error
			id, err = logManager.Add(models.LogEntry{
				Text:     text,
				ParentID: parentID,
			})
			if err != nil {
				return err
			}
			appState.Entries = logManager.Entries
			subEntryType = models.LogEntryViewType_Log
			return nil
		}
		err := appState.ChildSubmitState.Do(ctx, text, submit)
		if err != nil {
			return subEntryType, 0, err
		}
		return subEntryType, id, err
	}
	appState.OnUpdate = func(viewType models.LogEntryViewType, id int64, text string) error {
		if viewType != models.LogEntryViewType_Log {
			return nil
		}
		err := logManager.Update(id, models.LogEntryOptional{
			Text: &text,
		})
		if err != nil {
			return err
		}
		appState.Entries = logManager.Entries
		return nil
	}
	appState.OnRemoveFromGroup = func(viewType models.LogEntryViewType, id int64) error {
		if viewType != models.LogEntryViewType_Log {
			return nil
		}
		exp.LogGroupStore.Add(&exp.LogIDGroupMapping{
			LogID:   id,
			GroupID: 0,
		})
		appState.Entries = logManager.Entries
		return nil
	}

	appState.OnDelete = func(viewType models.LogEntryViewType, id int64) error {
		if viewType != models.LogEntryViewType_Log {
			return nil
		}
		if appState.ViewMode == app.ViewMode_Group {
			// delete from group
			exp.LogGroupStore.Add(&exp.LogIDGroupMapping{
				LogID:   id,
				GroupID: 0,
			})
			return nil
		}
		err := logManager.Delete(id)
		if err != nil {
			return err
		}
		appState.Entries = logManager.Entries
		return nil
	}
	appState.OnToggle = func(viewType models.LogEntryViewType, id int64) error {
		if viewType != models.LogEntryViewType_Log {
			return nil
		}
		foundEntry, err := logManager.Get(id)
		if err != nil {
			return err
		}

		done := !foundEntry.Data.Done
		var doneTime *time.Time
		if done {
			now := time.Now()
			doneTime = &now
		}
		err = logManager.Update(id, models.LogEntryOptional{
			Done:     &done,
			DoneTime: &doneTime,
		})
		if err != nil {
			return err
		}
		appState.Entries = logManager.Entries
		return nil
	}
	appState.OnPromote = func(viewType models.LogEntryViewType, id int64) error {
		if viewType != models.LogEntryViewType_Log {
			return nil
		}
		currentTime := time.Now().UnixMilli()
		err := logManager.Update(id, models.LogEntryOptional{
			AdjustedTopTime: &currentTime,
		})
		if err != nil {
			return err
		}
		appState.Entries = logManager.Entries
		return nil
	}
	appState.OnUpdateHighlight = func(viewType models.LogEntryViewType, id int64, highlightLevel int) {
		if viewType != models.LogEntryViewType_Log {
			return
		}
		logManager.Update(id, models.LogEntryOptional{
			HighlightLevel: &highlightLevel,
		})
		appState.Entries = logManager.Entries
	}

	appState.OnMove = func(id models.EntryIdentity, newParentID models.EntryIdentity) error {
		applog.Infof(context.TODO(), "moving: %v, %v to %v,%v", id.EntryType, id.ID, newParentID.EntryType, newParentID.ID)
		if id.EntryType == models.LogEntryViewType_Log && newParentID.EntryType == models.LogEntryViewType_Log {
			err := logManager.Move(id.ID, newParentID.ID)
			if err != nil {
				return err
			}
			appState.Entries = logManager.Entries
			return nil
		}
		if id.EntryType == models.LogEntryViewType_Log && newParentID.EntryType == models.LogEntryViewType_Group {
			// id -> group
			exp.LogGroupStore.Add(&exp.LogIDGroupMapping{
				LogID:   id.ID,
				GroupID: newParentID.ID,
			})
		}
		return nil
	}
	appState.OnAddNote = func(id int64, text string) error {
		err := logManager.AddNote(id, models.Note{
			Text: text,
		})
		if err != nil {
			return err
		}
		appState.Entries = logManager.Entries
		return nil
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
	appState.OnShowTop = func(id int64, text string, duration time.Duration) {
		// first make highlight level 5
		highlightLevel := 5
		err := logManager.Update(id, models.LogEntryOptional{
			HighlightLevel: &highlightLevel,
		})
		if err != nil {
			appState.StatusBar.Error = err.Error()
			return
		}
		appState.Entries = logManager.Entries

		// Send command to macOS app to show floating progress bar
		err = macos.SendTopCommand(id, text, duration)
		if err != nil {
			// Set error in status bar if command fails
			appState.StatusBar.Error = fmt.Sprintf("Failed to show top: %v", err)
		}
	}
	appState.OnToggleVisibility = func(id int64) error {
		targetEntry, err := logManager.Get(id)
		if err != nil {
			return err
		}

		// Toggle history inclusion state
		targetEntry.IncludeHistory = !targetEntry.IncludeHistory

		// Load children based on history inclusion setting
		ctx := context.Background()
		fullEntry, err := logManager.GetTree(ctx, id, targetEntry.IncludeHistory)
		if err != nil {
			return fmt.Errorf("load children: %w", err)
		}
		// Replace with loaded children (with or without history based on setting)
		targetEntry.Children = fullEntry.Children

		// Update the app state entries
		appState.Entries = logManager.Entries
		return nil
	}
	appState.OnToggleNotesDisplay = func(id int64) error {
		targetEntry, err := logManager.Get(id)
		if err != nil {
			return err
		}

		// Toggle notes display for this entry
		targetEntry.IncludeNotes = !targetEntry.IncludeNotes

		// Update the app state entries
		appState.Entries = logManager.Entries
		return nil
	}
	appState.OnToggleCollapsed = func(entryType models.LogEntryViewType, id int64) error {
		if entryType == models.LogEntryViewType_Log {
			err := logManager.ToggleCollapsed(id)
			if err != nil {
				return err
			}
			appState.Entries = logManager.Entries
			return nil
		} else if entryType == models.LogEntryViewType_Group {
			// Handle group collapse state in memory using thread-safe MutexMap
			appState.GroupCollapseState.Toggle(id)
			return nil
		}
		return nil
	}
	appState.Happening = app.HappeningState{
		LoadHappenings: func(ctx context.Context) ([]*models.Happening, error) {
			return logManager.HappeningManager.LoadHappenings(ctx)
		},
		AddHappening: func(ctx context.Context, content string) (*models.Happening, error) {
			return logManager.HappeningManager.AddHappening(ctx, content)
		},
		UpdateHappening: func(ctx context.Context, id int64, update *models.HappeningOptional) (*models.Happening, error) {
			return logManager.HappeningManager.UpdateHappening(ctx, id, update)
		},
		DeleteHappening: func(ctx context.Context, id int64) error {
			return logManager.HappeningManager.DeleteHappening(ctx, id)
		},
		Input: models.InputState{
			Focused: true,
		},
	}

	// Initialize SubmitState for happening with restore callback
	appState.Happening.SubmitState.SetOnRestore(func(content string) {
		// Append the failed content to existing happening input instead of replacing
		appState.Happening.Input.Value = appState.Happening.Input.Value + content
		appState.Happening.Input.CursorPosition = len([]rune(appState.Happening.Input.Value))
	})

	// Initialize sync.Once for state loading
	var loadStateOnce sync.Once

	// Helper function to load and refresh history
	loadHStatHistory := func(ctx context.Context) error {
		history, err := logManager.StateRecordingService.GetStateHistory(ctx, storage.GetStateHistoryOptions{
			Names: []string{human_state.HP_STATE_NAME},
			Days:  30,
		})
		if err != nil {
			return err
		}

		// Convert to HumanStateHistoryPoint
		appState.HumanState.History = make([]human_state.HumanStateHistoryPoint, len(history))
		for i, point := range history {
			appState.HumanState.History[i] = human_state.HumanStateHistoryPoint{
				Date:  point.Date,
				Score: point.Score,
			}
		}
		applog.Infof(ctx, "DEBUG Loaded H/P State history: %d points", len(history))
		return nil
	}

	loadHStat := func() {
		loadStateOnce.Do(func() {
			appState.Enqueue(func(ctx context.Context) error {
				state, err := logManager.StateRecordingService.GetState(ctx, human_state.HP_STATE_NAME)
				if err != nil {
					applog.Infof(ctx, "DEBUG Failed to fetch H/P State: %v", err)
					return err
				}
				appState.HumanState.HpScores = int(state.Score)
				applog.Infof(ctx, "DEBUG H/P State score: %f", state.Score)

				// Load state history
				if err := loadHStatHistory(ctx); err != nil {
					applog.Infof(ctx, "DEBUG Failed to fetch H/P State history: %v", err)
					return err
				}

				return nil
			})
		})
	}

	// Initialize human state
	appState.HumanState = &human_state.HumanState{
		HpScores:        0,
		FocusedBarIndex: -1,
		OnAdjustScore: func(delta int) error {
			// First update local score (already done in AdjustScore method)
			// Then record the state event
			ctx := context.Background()
			err := logManager.StateRecordingService.RecordStateEvent(ctx, human_state.HP_STATE_NAME, float64(delta))
			if err != nil {
				applog.Errorf(ctx, "Failed to record state event: %v", err)
				return err
			}
			applog.Infof(ctx, "DEBUG Recorded state event with delta: %d", delta)

			// Refresh history after recording the event
			if err := loadHStatHistory(ctx); err != nil {
				applog.Errorf(ctx, "Failed to refresh H/P State history: %v", err)
				// Don't return error, just log it
			}

			return nil
		},
		Enqueue:       appState.Enqueue,
		LoadStateOnce: loadHStat,
	}

	// Initialize sync.Once for learning materials loading
	var loadMaterialsOnce sync.Once

	loadLearningMaterials := func() {
		loadMaterialsOnce.Do(func() {
			appState.Learning.Loading = true
			appState.Enqueue(func(ctx context.Context) error {
				if appState.Learning.LoadMaterials == nil {
					appState.Learning.Error = "LoadMaterials is not set"
					return nil
				}
				materials, _, err := appState.Learning.LoadMaterials(ctx, 0, 10)
				if err != nil {
					appState.Learning.Error = err.Error()
					return err
				}
				// Update the state with loaded data
				appState.Learning.Loading = false
				appState.Learning.Materials = materials
				return nil
			})
		})
	}

	// Initialize learning state
	appState.Learning = app.LearningState{
		LoadMaterials: func(ctx context.Context, offset int, limit int) ([]*models.LearningMaterial, int64, error) {
			if services.LearningMaterials == nil {
				return nil, 0, fmt.Errorf("learning materials service not available")
			}
			return services.LearningMaterials.ListMaterials(ctx, offset, limit)
		},
		LoadMaterialsOnce: loadLearningMaterials,
	}

	// Initialize reading state
	appState.Reading = app.ReadingState{
		ContentCache: make(map[int]string),
		LoadContent: func(ctx context.Context, materialID int64, offset int, limit int) (string, int, int64, error) {
			if services.LearningMaterials == nil {
				return "", 0, 0, fmt.Errorf("learning materials service not available")
			}
			resp, err := services.LearningMaterials.GetMaterialContent(ctx, materialID, offset, limit)
			if err != nil {
				return "", 0, 0, err
			}
			return resp.Content, resp.TotalBytes, resp.LastOffset, nil
		},
		SavePosition: func(ctx context.Context, materialID int64, offset int64) error {
			if services.LearningMaterials == nil {
				return fmt.Errorf("learning materials service not available")
			}
			return services.LearningMaterials.UpdateReadingPosition(ctx, materialID, offset)
		},
	}

	if group {
		appState.ViewMode = app.ViewMode_Group
	} else if humanStat {
		loadHStat()
		appState.Routes.Push(app.HumanStateRoute())
	} else if learning {
		loadLearningMaterials()
		appState.Routes.Push(app.LearningRoute())
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

func handleTool(args []string) error {
	return tool.Handle(args)
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
