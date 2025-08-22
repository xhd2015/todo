package app

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
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

type State struct {
	Entries models.LogEntryViews

	Input               models.InputState
	SelectedEntryID     int64
	SelectFromSource    SelectedSource
	LastSelectedEntryID int64
	SelectedEntryMode   SelectedEntryMode
	SelectedInputState  models.InputState
	ChildInputState     models.InputState

	SelectedDeleteConfirmButton int

	// in ZenMode, only show highlighted and
	// unfinished entries
	ZenMode bool

	SelectedActionIndex int

	Routes Routes

	ShowHistory bool // Whether to show historical (done) todos from before today

	// Search functionality
	SearchQuery    string // Current search query (without the ? prefix)
	IsSearchActive bool   // Whether search mode is active

	// Pagination
	SliceStart int // Starting index for the slice of entries to display

	// Cut/Paste functionality
	CuttingEntryID int64 // ID of the entry currently being cut (0 if none)

	Quit func()

	Refresh func()

	OnAdd             func(string)
	OnAddChild        func(parentID int64, text string) (int64, error)
	OnUpdate          func(id int64, text string)
	OnDelete          func(id int64)
	OnToggle          func(id int64)
	OnPromote         func(id int64)
	OnUpdateHighlight func(id int64, highlightLevel int)
	OnMove            func(id int64, newParentID int64)

	OnAddNote    func(id int64, text string) error
	OnUpdateNote func(entryID int64, noteID int64, text string)
	OnDeleteNote func(entryID int64, noteID int64)

	RefreshEntries     func(ctx context.Context) error                     // Callback to refresh entries when ShowHistory changes
	OnShowTop          func(id int64, text string, duration time.Duration) // Callback to show todo in macOS floating bar
	OnToggleVisibility func(id int64)                                      // Callback to toggle visibility of all children including history

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

// IsDescendant checks if potentialChild is a descendant of potentialParent
func (state *State) IsDescendant(potentialChild int64, potentialParent int64) bool {
	if potentialChild == potentialParent {
		return true
	}

	for _, entry := range state.Entries {
		if entry.Data.ID == potentialChild {
			if entry.Data.ParentID == potentialParent {
				return true
			}
			if entry.Data.ParentID != 0 {
				return state.IsDescendant(entry.Data.ParentID, potentialParent)
			}
			break
		}
	}
	return false
}

func (state *State) Deselect() {
	state.SelectedEntryID = 0
	state.SelectFromSource = SelectedSource_Default
}

func (state *State) Select(id int64) {
	state.SelectedEntryID = id
	state.SelectFromSource = SelectedSource_Default
}

// Enqueue schedules an action to run in a goroutine and tracks its status
func (state *State) Enqueue(action func(ctx context.Context) error) {
	state.actionQueueMutex.Lock()
	state.activeActions++
	state.actionQueueMutex.Unlock()

	go func() {
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
		dom.H1(dom.DivProps{}, dom.Text("TODO List", styles.Style{
			Bold:        true,
			BorderColor: "orange",
		})),

		func() *dom.Node {
			if len(state.Routes) == 0 {
				return MainPage(state, window)
			} else {
				return RenderRoute(state, state.Routes.Last())
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
		func() *dom.Node {
			// Build status bar nodes
			var nodes []*dom.Node

			// Left side: dot, storage, error, requesting
			nodes = append(nodes, dom.Text("•", styles.Style{
				Bold:  true,
				Color: colors.GREEN_SUCCESS,
			}))
			if state.StatusBar.Storage != "" {
				nodes = append(nodes, dom.Text(state.StatusBar.Storage, styles.Style{
					Bold:  true,
					Color: colors.GREY_TEXT,
				}))
			}
			if state.StatusBar.Error != "" {
				nodes = append(nodes, dom.Text("  "+state.StatusBar.Error, styles.Style{
					Bold:  true,
					Color: colors.RED_ERROR,
				}))
			}
			if state.Requesting() {
				nodes = append(nodes, dom.Text("  •", styles.Style{
					Bold:  true,
					Color: colors.GREEN_SUCCESS,
				}))
				nodes = append(nodes, dom.Text("Request...", styles.Style{
					Bold:  true,
					Color: colors.GREEN_SUCCESS,
				}))
			}

			// Spacer to push modes to the right
			hasRightContent := state.ZenMode || state.ShowHistory
			if hasRightContent {
				nodes = append(nodes, dom.Spacer())

				// Right side: modes
				if state.ZenMode {
					nodes = append(nodes, dom.Text("zen", styles.Style{
						Bold:  true,
						Color: colors.GREY_TEXT,
					}))
				}
				if state.ShowHistory {
					if state.ZenMode {
						nodes = append(nodes, dom.Text(" ", styles.Style{}))
					}
					nodes = append(nodes, dom.Text("history", styles.Style{
						Bold:  true,
						Color: colors.GREY_TEXT,
					}))
				}
			}

			return dom.Div(dom.DivProps{Width: UIWidth}, nodes...)
		}(),
	)
}
