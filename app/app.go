package app

import (
	"time"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models"
)

const (
	CtrlCExitDelayMs = 1000
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

	EnteredEntryID int64

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

	OnAddNote    func(id int64, text string)
	OnUpdateNote func(entryID int64, noteID int64, text string)
	OnDeleteNote func(entryID int64, noteID int64)

	OnRefreshEntries func() // Callback to refresh entries when ShowHistory changes

	LastCtrlC time.Time
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
				if state.EnteredEntryID > 0 {
					state.EnteredEntryID = 0
				}
			}
		},
	},
		dom.H1(dom.DivProps{}, dom.Text("TODO List", styles.Style{
			Bold:        true,
			BorderColor: "orange",
		})),

		func() *dom.Node {
			if state.EnteredEntryID == 0 {
				return MainPage(state, window)
			} else {
				return DetailPage(state, state.EnteredEntryID)
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
	)
}
