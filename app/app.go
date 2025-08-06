package app

import (
	"fmt"
	"strings"
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

type State struct {
	Entries models.LogEntryViews

	Input              models.InputState
	SelectedEntryID    int64
	SelectedEntryMode  SelectedEntryMode
	SelectedInputState models.InputState
	ChildInputState    models.InputState

	SelectedDeleteConfirmButton int

	SelectedActionIndex int

	EnteredEntryID int64

	ShowHistory bool // Whether to show historical (done) todos from before today

	// Search functionality
	SearchQuery    string // Current search query (without the ? prefix)
	IsSearchActive bool   // Whether search mode is active

	Quit func()

	Refresh func()

	OnAdd             func(string)
	OnAddChild        func(parentID int64, text string)
	OnUpdate          func(id int64, text string)
	OnDelete          func(id int64)
	OnToggle          func(id int64)
	OnPromote         func(id int64)
	OnUpdateHighlight func(id int64, highlightLevel int)

	OnAddNote func(id int64, text string)

	OnRefreshEntries func() // Callback to refresh entries when ShowHistory changes

	LastCtrlC time.Time
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

func MainPage(state *State, window *dom.Window) *dom.Node {
	height := window.Height
	availableHeight := height - 5 - len(state.Entries)
	if availableHeight < 3 {
		availableHeight = 3
	}
	var brs []*dom.Node
	if availableHeight > 3 {
		brs = make([]*dom.Node, availableHeight-3)
		for i := range brs {
			brs[i] = dom.Br()
		}
	}

	// Render the tree of entries
	children := RenderEntryTree(state)
	return dom.Fragment(
		dom.Ul(dom.DivProps{}, children...),
		dom.Fragment(brs...),
		// input
		func() *dom.Node {
			placeholder := "add todo"
			if state.IsSearchActive {
				placeholder = "search todos (ESC to exit search)"
			}

			return SearchInput(InputProps{
				Placeholder: placeholder,
				State:       &state.Input,
				onEnter: func(s string) bool {
					if strings.TrimSpace(s) == "" {
						return false
					}

					// Handle search mode
					if state.IsSearchActive {
						// In search mode, enter just exits search
						state.IsSearchActive = false
						state.SearchQuery = ""
						return true
					}

					if s == "exit" || s == "quit" || s == "q" {
						state.Quit()
						return true
					}

					// Handle special commands starting with /
					if strings.HasPrefix(s, "/") {
						switch s {
						case "/history":
							// Toggle ShowHistory and refresh entries
							state.ShowHistory = !state.ShowHistory
							if state.OnRefreshEntries != nil {
								state.OnRefreshEntries()
							}
							return true
						case "/reload":
							if state.OnRefreshEntries != nil {
								state.OnRefreshEntries()
							}
							return true
						default:
							// Unknown command, do nothing
							return true
						}
					}

					state.OnAdd(s)
					return true
				},
				onSearchChange: func(query string) {
					state.SearchQuery = query
				},
				onSearchActivate: func() {
					state.IsSearchActive = true
				},
				onSearchDeactivate: func() {
					state.IsSearchActive = false
					state.SearchQuery = ""
				},
			})
		}(),
	)
}

func DetailPage(state *State, id int64) *dom.Node {
	item := state.Entries.Get(id)
	if item == nil {
		return dom.Text(fmt.Sprintf("not found: %d", id))
	}

	return dom.Div(dom.DivProps{
		OnKeyDown: func(d *dom.DOMEvent) {
			keyEvent := d.KeydownEvent
			switch keyEvent.KeyType {
			case dom.KeyTypeEsc:
				state.EnteredEntryID = 0
			}
		},
	},
		dom.Text(item.Data.Text),

		dom.H1(dom.DivProps{}, dom.Text("Notes")),

		func() *dom.Node {
			notes := item.Notes

			if len(notes) == 0 {
				return dom.Fragment(dom.Text("No notes"), dom.Br())
			}
			var children []*dom.Node
			for _, note := range notes {
				children = append(children, dom.Li(dom.ListItemProps{}, dom.Text(note.Data.Text)))
			}
			return dom.Ul(dom.DivProps{}, children...)
		}(),

		SearchInput(InputProps{
			Placeholder: "add note",
			State:       item.DetailPage.InputState,
			onEnter: func(value string) bool {
				state.OnAddNote(item.Data.ID, value)
				return true
			},
		}),
	)
}
