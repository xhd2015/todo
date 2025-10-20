package app

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/app/human_state"
	"github.com/xhd2015/todo/models"
)

const (
	CtrlCExitDelayMs = 1000
	UIWidth          = 50 // Shared width for status bar and input components
)

type SelectedEntryMode int

const (
	SelectedEntryMode_Default = iota
	SelectedEntryMode_Editing
	SelectedEntryMode_ShowActions
	SelectedEntryMode_DeleteConfirm
	SelectedEntryMode_AddingChild
)

type SelectedSource int

const (
	SelectedSource_Default SelectedSource = iota
	SelectedSource_Search
	SelectedSource_NavigateByKey
)

type ViewMode int

const (
	ViewMode_Default ViewMode = iota
	ViewMode_Group
)

type HappeningState struct {
	Loading    bool
	Happenings []*models.Happening
	Error      string
	Input      models.InputState

	FocusedItemID int64

	// Edit/Delete state
	EditingItemID       int64
	EditInputState      models.InputState
	DeletingItemID      int64
	DeleteConfirmButton int // 0 = Delete, 1 = Cancel

	LoadHappenings  func(ctx context.Context) ([]*models.Happening, error)
	AddHappening    func(ctx context.Context, content string) (*models.Happening, error)
	UpdateHappening func(ctx context.Context, id int64, update *models.HappeningOptional) (*models.Happening, error)
	DeleteHappening func(ctx context.Context, id int64) error
}

type State struct {
	Entries models.LogEntryViews

	Input               models.InputState
	SelectedEntry       models.EntryIdentity
	LastSelectedEntry   models.EntryIdentity
	SelectedNoteID      int64 // ID of the selected note (0 if none)
	SelectedNoteEntryID int64 // ID of the entry that owns the selected note
	SelectFromSource    SelectedSource
	SelectedEntryMode   SelectedEntryMode
	SelectedInputState  models.InputState
	ChildInputState     models.InputState

	SelectedDeleteConfirmButton int

	// in ZenMode, only show highlighted and
	// unfinished entries
	ZenMode bool

	SelectedActionIndex int

	Routes Routes

	// Happening functionality
	Happening HappeningState

	// Human state functionality
	HumanState *human_state.HumanState

	ShowHistory bool // Whether to show historical (done) todos from before today
	ShowNotes   bool // Whether to show all notes globally
	ExpandAll   bool // Whether to expand all entries, ignoring individual collapse flags

	// Search functionality
	SearchQuery    string // Current search query (without the ? prefix)
	IsSearchActive bool   // Whether search mode is active

	// Pagination
	SliceStart int // Starting index for the slice of entries to display

	// Cut/Paste functionality
	CuttingEntry models.EntryIdentity // ID of the entry currently being cut (0 if none)

	// Focused mode functionality
	FocusedEntry models.EntryIdentity // ID of the entry currently focused on (0 if none)

	// View mode functionality
	ViewMode ViewMode // Current view mode (default or group)

	Quit func()

	Refresh func()

	OnAdd             func(viewType models.LogEntryViewType, text string) error
	OnAddChild        func(viewType models.LogEntryViewType, parentID int64, text string) (int64, error)
	OnUpdate          func(viewType models.LogEntryViewType, id int64, text string) error
	OnDelete          func(viewType models.LogEntryViewType, id int64) error
	OnToggle          func(viewType models.LogEntryViewType, id int64) error
	OnPromote         func(viewType models.LogEntryViewType, id int64) error
	OnUpdateHighlight func(viewType models.LogEntryViewType, id int64, highlightLevel int)
	OnMove            func(id models.EntryIdentity, newParentID models.EntryIdentity) error

	OnAddNote    func(id int64, text string) error
	OnUpdateNote func(entryID int64, noteID int64, text string)
	OnDeleteNote func(entryID int64, noteID int64)

	RefreshEntries       func(ctx context.Context) error                         // Callback to refresh entries when ShowHistory changes
	OnShowTop            func(id int64, text string, duration time.Duration)     // Callback to show todo in macOS floating bar
	OnToggleVisibility   func(id int64) error                                    // Callback to toggle visibility of all children including history
	OnToggleNotesDisplay func(id int64) error                                    // Callback to toggle notes display for entry and its subtree
	OnToggleCollapsed    func(entryType models.LogEntryViewType, id int64) error // Callback to toggle collapsed state for entry

	LastCtrlC time.Time

	// Action queue for tracking ongoing operations
	actionQueueMutex sync.RWMutex
	activeActions    int

	StatusBar StatusBar
}

type StatusBar struct {
	Error   string
	Storage string
}

type IDs []int64

func (ids *IDs) Pop() {
	*ids = (*ids)[:len(*ids)-1]
}

func (ids *IDs) Push(id int64) {
	*ids = append(*ids, id)
}

func (ids IDs) SetLast(id int64) {
	ids[len(ids)-1] = id
}

func (ids IDs) Last() int64 {
	return ids[len(ids)-1]
}

func (state *State) ClearSearch() {
	state.IsSearchActive = false
	state.SearchQuery = ""
	state.Input.Reset()
}

// ResetAllChildrenVisibility resets all IncludeHistory states to false
// This is used when /history is toggled off to reset all 'v' command states
func (state *State) ResetAllChildrenVisibility() {
	var resetVisibility func(entry *models.LogEntryView)
	resetVisibility = func(entry *models.LogEntryView) {
		entry.IncludeHistory = false
		for _, child := range entry.Children {
			resetVisibility(child)
		}
	}

	for _, entry := range state.Entries {
		resetVisibility(entry)
	}
}

// IsDescendant checks if potentialChild is a descendant of potentialParent
func (state *State) IsDescendant(potentialChild models.EntryIdentity, potentialParent models.EntryIdentity) bool {
	if potentialChild == potentialParent {
		return true
	}

	for _, entry := range state.Entries {
		if entry.SameIdentity(potentialChild) {
			if potentialParent.EntryType == models.LogEntryViewType_Log && entry.Data.ParentID == potentialParent.ID {
				return true
			}
			if entry.Data.ParentID != 0 {
				return state.IsDescendant(models.EntryIdentity{
					EntryType: models.LogEntryViewType_Log,
					ID:        entry.Data.ParentID,
				}, potentialParent)
			}
			break
		}
	}
	return false
}

func (state *State) Deselect() {
	state.SelectedEntry = models.EntryIdentity{}
	state.SelectedNoteID = 0
	state.SelectedNoteEntryID = 0
	state.SelectFromSource = SelectedSource_Default
}

func (state *State) Select(entyType models.LogEntryViewType, id int64) {
	state.SelectedEntry = models.EntryIdentity{
		EntryType: entyType,
		ID:        id,
	}
	state.SelectedNoteID = 0
	state.SelectedNoteEntryID = 0
	state.SelectFromSource = SelectedSource_Default
}

func (state *State) SelectNote(noteID int64, entryID int64) {
	state.SelectedEntry = models.EntryIdentity{}
	state.SelectedNoteID = noteID
	state.SelectedNoteEntryID = entryID
	state.SelectFromSource = SelectedSource_Default
}

const _REFRESH_DELAY = 200 * time.Millisecond

// Enqueue schedules an action to run in a goroutine and tracks its status
func (state *State) Enqueue(action func(ctx context.Context) error) {
	state.actionQueueMutex.Lock()
	state.activeActions++
	state.actionQueueMutex.Unlock()

	go func() {
		begin := time.Now()
		defer func() {
			e := recover()
			if e != nil {
				stack := debug.Stack()
				state.StatusBar.Error = fmt.Sprintf("panic: %v\n%s", e, string(stack))
			}
			state.actionQueueMutex.Lock()
			state.activeActions--
			state.actionQueueMutex.Unlock()

			// Trigger refresh to update UI
			if state.Refresh != nil {
				// wait for 200ms to avoid flickering
				elapsed := time.Since(begin)
				if elapsed < _REFRESH_DELAY {
					time.Sleep(_REFRESH_DELAY - elapsed)
				}
				state.Refresh()
			}
		}()

		ctx := context.Background()
		if err := action(ctx); err != nil {
			// Set error in status bar
			state.actionQueueMutex.Lock()
			state.StatusBar.Error = err.Error()
			state.actionQueueMutex.Unlock()
		}
	}()
}

// Requesting returns true if there are ongoing actions
func (state *State) Requesting() bool {
	state.actionQueueMutex.RLock()
	defer state.actionQueueMutex.RUnlock()
	return state.activeActions > 0
}

func App(state *State, window *dom.Window) *dom.Node {
	return dom.Div(dom.DivProps{
		OnKeyDown: func(event *dom.DOMEvent) {
			keyEvent := event.KeydownEvent
			if keyEvent == nil {
				return
			}
			switch keyEvent.KeyType {
			case dom.KeyTypeCtrlC:
				if time.Since(state.LastCtrlC) < time.Millisecond*CtrlCExitDelayMs {
					state.Quit()
					return
				}
				state.LastCtrlC = time.Now()

				go func() {
					time.Sleep(time.Millisecond * CtrlCExitDelayMs)
					state.Refresh()
				}()
			case dom.KeyTypeEsc:
				if len(state.Routes) > 0 {
					state.Routes.Pop()
				}
			}
		},
	},
		func() *dom.Node {
			title := "TODO List"
			if len(state.Routes) > 0 {
				last := state.Routes.Last()
				switch last.Type {
				case RouteType_Main:
					title = "Main"
				case RouteType_Detail:
					title = "Detail"
				case RouteType_Config:
					title = "Config"
				case RouteType_HappeningList:
					title = "Happenings"
				case RouteType_HumanState:
					title = "Human States"
				case RouteType_Help:
					title = "Help"
				}
			}
			return dom.H1(dom.DivProps{}, dom.Text(title, styles.Style{
				Bold:        true,
				BorderColor: "orange",
			}))
		}(),
		func() *dom.Node {
			if false {
				return nil
			}
			if len(state.Routes) == 0 {
				return MainPage(state, window)
			} else {
				return RenderRoute(state, state.Routes.Last(), window)
			}
		}(),
		func() *dom.Node {
			if time.Since(state.LastCtrlC) < time.Millisecond*CtrlCExitDelayMs {
				return dom.Text("press Ctrl-C again to exit", styles.Style{
					Bold:  true,
					Color: "1",
				})
			}
			return dom.Text("type 'exit','quit' or 'q' to exit")
		}(),
		AppStatusBar(state),
	)
}
