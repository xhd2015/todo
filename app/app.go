package app

import (
	"time"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/todo/models"
)

const (
	CtrlCExitDelayMs = 1000
)

type State struct {
	Entries []*models.EntryView

	Input                models.InputState
	SelectedEntryIndex   int
	SelectedEntryEditing bool
	SelectedInputState   models.InputState

	SelectedShowDeleteConfirm   bool
	SelectedDeleteConfirmButton int

	EnteredEntryIndex int

	Quit func()

	Refresh func()

	OnAdd    func(string)
	OnUpdate func(id int64, text string)
	OnDelete func(id int64)

	OnAddNote func(id int64, text string)

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

		var children []*dom.Node
		for i, item := range state.Entries {
			if state.SelectedEntryEditing && state.SelectedEntryIndex == i {
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
							state.SelectedEntryEditing = false
						case "enter":
							state.OnUpdate(item.ID, state.SelectedInputState.Value)
							state.SelectedEntryEditing = false
						}
					},
				}))
				continue
			}
			children = append(children, dom.Li(dom.ListItemProps{
				Focusable: dom.Focusable(true),
				Selected:  state.SelectedEntryIndex == i,
				Focused:   !state.SelectedShowDeleteConfirm && state.SelectedEntryIndex == i,
				OnFocus: func() {
					state.SelectedEntryIndex = i
				},
				OnBlur: func() {
					state.SelectedEntryIndex = -1
				},
				Text: item.Text,
				OnKeyDown: func(e *dom.DOMEvent) {
					switch e.Key {
					case "e":
						state.SelectedEntryEditing = true
						state.SelectedInputState.Value = item.Text
						state.SelectedInputState.Focused = true
						state.SelectedInputState.CursorPosition = len(item.Text) + 1
					case "d":
						state.SelectedShowDeleteConfirm = true
						state.SelectedDeleteConfirmButton = 0
					case "enter":
						if state.SelectedShowDeleteConfirm {
							state.SelectedShowDeleteConfirm = false
							return
						}
						state.EnteredEntryIndex = i

						item.DetailPage.InputState.Value = ""
						item.DetailPage.InputState.Focused = true
						item.DetailPage.InputState.CursorPosition = 0
					case "esc":
						state.SelectedShowDeleteConfirm = false
					case "up", "down":
						state.SelectedShowDeleteConfirm = false
					case "left", "right":
						delta := 1
						if e.Key == "left" {
							delta = -1
						}
						state.SelectedDeleteConfirmButton += delta
						if state.SelectedDeleteConfirmButton < 0 {
							state.SelectedDeleteConfirmButton = 1
						}
						if state.SelectedDeleteConfirmButton > 1 {
							state.SelectedDeleteConfirmButton = 0
						}
					}
				},
			}))

			if state.SelectedShowDeleteConfirm && state.SelectedEntryIndex == i {
				children = append(children, ConfirmDialog(ConfirmDialogProps{
					SelectedButton: state.SelectedDeleteConfirmButton,
					PromptText:     "Delete todo?",
					DeleteText:     "[Delete]",
					CancelText:     "[Cancel]",
					OnDelete: func() {
						state.OnDelete(item.ID)
						// move selection
						if state.SelectedEntryIndex > len(state.Entries)-1 {
							state.SelectedEntryIndex = len(state.Entries) - 1
						}
						state.SelectedShowDeleteConfirm = false
					},
					OnCancel: func() {
						state.SelectedShowDeleteConfirm = false
					},
					OnNavigateRight: func() {
						state.SelectedDeleteConfirmButton = 1
					},
					OnNavigateLeft: func() {
						state.SelectedDeleteConfirmButton = 0
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
				onEnter:     state.OnAdd,
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
			dom.Text(item.Text),

			dom.H1(dom.DivProps{}, dom.Text("Notes")),

			func() *dom.Node {
				notes := item.Notes

				if len(notes) == 0 {
					return dom.Fragment(dom.Text("No notes"), dom.Br())
				}
				var children []*dom.Node
				for _, note := range notes {
					children = append(children, dom.Li(dom.ListItemProps{
						Text: note.Text,
					}))
				}
				return dom.Ul(dom.DivProps{}, children...)
			}(),

			BindInput(InputProps{
				Placeholder: "add note",
				State:       item.DetailPage.InputState,
				onEnter: func(value string) {
					state.OnAddNote(item.ID, value)
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
		dom.H1(dom.DivProps{}, dom.Text("TODO List", dom.Style{
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
				return dom.Text("press Ctrl-C again to exit", dom.Style{
					Bold:  true,
					Color: "1",
				})
			}
			return nil
		}(),
	)
}
