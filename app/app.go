package app

import (
	"strings"
	"time"

	"github.com/xhd2015/go-dom-tui/colors"
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
	Entries []*models.EntryView

	Input              models.InputState
	SelectedEntryIndex int
	SelectedEntryMode  SelectedEntryMode
	SelectedInputState models.InputState
	ChildInputState    models.InputState

	SelectedDeleteConfirmButton int

	SelectedActionIndex int

	EnteredEntryIndex int

	ShowHistory bool // Whether to show historical (done) todos from before today

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
	mainPage := func() *dom.Node {
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

		// Build a flat list of all entries with their depths for rendering
		type EntryWithDepth struct {
			Entry *models.EntryView
			Index int
			Depth int
		}

		var flatEntries []EntryWithDepth
		entryIndex := 0

		var addEntryRecursive func(entry *models.EntryView, depth int)
		addEntryRecursive = func(entry *models.EntryView, depth int) {
			flatEntries = append(flatEntries, EntryWithDepth{
				Entry: entry,
				Index: entryIndex,
				Depth: depth,
			})
			entryIndex++

			// Add children recursively
			for _, child := range entry.Children {
				addEntryRecursive(child, depth+1)
			}
		}

		// Add top-level entries (ParentID == 0)
		for _, entry := range state.Entries {
			if entry.Data.ParentID == 0 {
				addEntryRecursive(entry, 0)
			}
		}

		var children []*dom.Node
		for _, entryWithDepth := range flatEntries {
			item := entryWithDepth.Entry
			i := entryWithDepth.Index
			depth := entryWithDepth.Depth
			isSelected := state.SelectedEntryIndex == i

			if state.SelectedEntryMode == SelectedEntryMode_Editing && isSelected {
				children = append(children, dom.Input(dom.InputProps{
					Value:          state.SelectedInputState.Value,
					Focused:        state.SelectedInputState.Focused,
					CursorPosition: state.SelectedInputState.CursorPosition,
					OnCursorMove: func(delta int, seek int) {
						state.SelectedInputState.CursorPosition += delta
					},
					OnChange: func(value string) {
						state.SelectedInputState.Value = value
					},
					OnKeyDown: func(e *dom.DOMEvent) {
						switch e.Key {
						case "up", "down":
							e.PreventDefault()
						case "esc":
							state.SelectedEntryMode = SelectedEntryMode_Default
						case "enter":
							state.OnUpdate(item.Data.ID, state.SelectedInputState.Value)
							state.SelectedEntryMode = SelectedEntryMode_Default
						}
					},
				}))
				continue
			}

			// Always render the TodoItem
			children = append(children, TodoItem(TodoItemProps{
				Item:       item,
				Index:      i,
				Depth:      depth,
				IsSelected: isSelected,
				State:      state,
			}))

			// Render child input box under the item when in AddingChild mode
			if state.SelectedEntryMode == SelectedEntryMode_AddingChild && isSelected {
				children = append(children, dom.Input(dom.InputProps{
					Placeholder:    "add child todo",
					Value:          state.ChildInputState.Value,
					Focused:        state.ChildInputState.Focused,
					CursorPosition: state.ChildInputState.CursorPosition,
					OnCursorMove: func(delta int, seek int) {
						state.ChildInputState.CursorPosition += delta
					},
					OnChange: func(value string) {
						state.ChildInputState.Value = value
					},
					OnKeyDown: func(e *dom.DOMEvent) {
						switch e.Key {
						case "up", "down":
							e.PreventDefault()
						case "esc":
							state.SelectedEntryMode = SelectedEntryMode_Default
						case "enter":
							if strings.TrimSpace(state.ChildInputState.Value) != "" {
								state.OnAddChild(item.Data.ID, state.ChildInputState.Value)
								state.ChildInputState.Value = ""
								state.ChildInputState.CursorPosition = 0
							}
							state.SelectedEntryMode = SelectedEntryMode_Default
						}
					},
				}))
			}

			if state.SelectedEntryMode == SelectedEntryMode_DeleteConfirm && isSelected {
				children = append(children, ConfirmDialog(ConfirmDialogProps{
					SelectedButton: state.SelectedDeleteConfirmButton,
					PromptText:     "Delete todo?",
					DeleteText:     "[Delete]",
					CancelText:     "[Cancel]",
					OnDelete: func() {
						state.OnDelete(item.Data.ID)
						// move selection
						if state.SelectedEntryIndex > len(state.Entries)-1 {
							state.SelectedEntryIndex = len(state.Entries) - 1
						}
						state.SelectedEntryMode = SelectedEntryMode_Default
					},
					OnCancel: func() {
						state.SelectedEntryMode = SelectedEntryMode_Default
					},
					OnNavigateRight: func() {
						state.SelectedDeleteConfirmButton = 1
					},
					OnNavigateLeft: func() {
						state.SelectedDeleteConfirmButton = 0
					},
				}))
			}

			if state.SelectedEntryMode == SelectedEntryMode_ShowActions && isSelected {
				children = append(children, Menu(MenuProps{
					Title:         "Promote",
					SelectedIndex: state.SelectedActionIndex,
					OnSelect: func(index int) {
						state.SelectedActionIndex = index
					},
					Items: []MenuItem{
						{Text: "Promote", OnSelect: func() {
							state.OnPromote(item.Data.ID)
							state.SelectedEntryMode = SelectedEntryMode_Default

							// set selected to bottom
							state.SelectedEntryIndex = len(state.Entries) - 1
						}},
						{Text: "No Highlight", OnSelect: func() {
							state.OnUpdateHighlight(item.Data.ID, 0)
							state.SelectedEntryMode = SelectedEntryMode_Default
						}},
						{Text: "Highlight-1", Color: colors.DARK_RED_1, OnSelect: func() {
							state.OnUpdateHighlight(item.Data.ID, 1)
							state.SelectedEntryMode = SelectedEntryMode_Default
						}},
						{Text: "Highlight-2", Color: colors.DARK_RED_3, OnSelect: func() {
							state.OnUpdateHighlight(item.Data.ID, 2)
							state.SelectedEntryMode = SelectedEntryMode_Default
						}},
						{Text: "Highlight-3", Color: colors.DARK_RED_4, OnSelect: func() {
							state.OnUpdateHighlight(item.Data.ID, 3)
							state.SelectedEntryMode = SelectedEntryMode_Default
						}},
						{Text: "Highlight-4", Color: colors.DARK_RED_5, OnSelect: func() {
							state.OnUpdateHighlight(item.Data.ID, 4)
							state.SelectedEntryMode = SelectedEntryMode_Default
						}},
						{Text: "Highlight-5", Color: colors.DARK_RED_5, OnSelect: func() {
							state.OnUpdateHighlight(item.Data.ID, 5)
							state.SelectedEntryMode = SelectedEntryMode_Default
						}},
					},
					OnKeyDown: func(e *dom.DOMEvent) {
						switch e.Key {
						case "up", "down":
							e.PreventDefault()
						}
					},
					OnDismiss: func() {
						state.SelectedEntryMode = SelectedEntryMode_Default
					},
				}))
			}
		}
		return dom.Fragment(
			dom.Ul(dom.DivProps{}, children...),
			dom.Fragment(brs...),
			// input
			BindInput(InputProps{
				Placeholder: "add todo",
				State:       &state.Input,
				onEnter: func(s string) bool {
					if strings.TrimSpace(s) == "" {
						return false
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
						default:
							// Unknown command, do nothing
							return true
						}
					}

					state.OnAdd(s)
					return true
				},
			}),
		)
	}

	detailPage := func(item *models.EntryView) *dom.Node {
		return dom.Div(dom.DivProps{
			OnKeyDown: func(d *dom.DOMEvent) {
				switch d.Key {
				case "esc":
					state.EnteredEntryIndex = -1
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

			BindInput(InputProps{
				Placeholder: "add note",
				State:       item.DetailPage.InputState,
				onEnter: func(value string) bool {
					state.OnAddNote(item.Data.ID, value)
					return true
				},
			}),
		)
	}

	return dom.Div(dom.DivProps{
		OnKeyDown: func(event *dom.DOMEvent) {
			switch event.Key {
			case "ctrl+c":
				if time.Since(state.LastCtrlC) < time.Millisecond*CtrlCExitDelayMs {
					state.Quit()
					return
				}
				state.LastCtrlC = time.Now()

				go func() {
					time.Sleep(time.Millisecond * CtrlCExitDelayMs)
					state.Refresh()
				}()
			case "esc":
				if state.EnteredEntryIndex >= 0 {
					state.EnteredEntryIndex = -1
				}
			}
		},
	},
		dom.H1(dom.DivProps{}, dom.Text("TODO List", styles.Style{
			Bold:        true,
			BorderColor: "orange",
		})),

		func() *dom.Node {
			if state.EnteredEntryIndex < 0 {
				return mainPage()
			} else {
				return detailPage(state.Entries[state.EnteredEntryIndex])
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
