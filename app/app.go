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
)

type State struct {
	Entries []*models.EntryView

	Input              models.InputState
	SelectedEntryIndex int
	SelectedEntryMode  SelectedEntryMode
	SelectedInputState models.InputState

	SelectedDeleteConfirmButton int

	SelectedActionIndex int

	EnteredEntryIndex int

	Quit func()

	Refresh func()

	OnAdd             func(string)
	OnUpdate          func(id int64, text string)
	OnDelete          func(id int64)
	OnToggle          func(id int64)
	OnPromote         func(id int64)
	OnUpdateHighlight func(id int64, highlightLevel int)

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

			children = append(children, dom.Li(dom.ListItemProps{
				Focusable: dom.Focusable(true),
				Selected:  isSelected,
				Focused:   state.SelectedEntryMode == SelectedEntryMode_Default && isSelected,
				ItemPrefix: dom.String(func() string {
					if item.Data.Done {
						return "✓ "
					}
					return "• "
				}()),
				OnFocus: func() {
					state.SelectedEntryIndex = i
				},
				OnBlur: func() {
					state.SelectedEntryIndex = -1
				},
				OnKeyDown: func(e *dom.DOMEvent) {
					switch e.Key {
					case "e":
						state.SelectedEntryMode = SelectedEntryMode_Editing
						state.SelectedInputState.Value = item.Data.Text
						state.SelectedInputState.Focused = true
						state.SelectedInputState.CursorPosition = len(item.Data.Text) + 1
					case "d":
						state.SelectedEntryMode = SelectedEntryMode_DeleteConfirm
						state.SelectedDeleteConfirmButton = 0
					case "enter":
						if state.SelectedEntryMode == SelectedEntryMode_DeleteConfirm {
							state.SelectedEntryMode = SelectedEntryMode_Default
							return
						}
						state.EnteredEntryIndex = i

						item.DetailPage.InputState.Value = ""
						item.DetailPage.InputState.Focused = true
						item.DetailPage.InputState.CursorPosition = 0
					case "esc":
						state.SelectedEntryMode = SelectedEntryMode_Default
					case "up", "down":
						state.SelectedEntryMode = SelectedEntryMode_Default
					case "left", "right":
						if state.SelectedEntryMode == SelectedEntryMode_DeleteConfirm {
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
						} else if state.SelectedEntryMode == SelectedEntryMode_Default {
							if e.Key == "right" {
								// show actions
								state.SelectedEntryMode = SelectedEntryMode_ShowActions
							}
						}
					case " ":
						// toggle status
						state.OnToggle(item.Data.ID)
					}
				},
			}, dom.Text(item.Data.Text, styles.Style{
				Color: func() string {
					if isSelected {
						return colors.GREEN_SUCCESS
					} else if item.Data.HighlightLevel > 4 {
						return colors.DARK_RED_5
					} else if item.Data.HighlightLevel > 3 {
						return colors.DARK_RED_4
					} else if item.Data.HighlightLevel > 2 {
						return colors.DARK_RED_3
					} else if item.Data.HighlightLevel > 1 {
						return colors.DARK_RED_2
					} else if item.Data.HighlightLevel == 1 {
						return colors.DARK_RED_1
					} else {
						return ""
					}
				}(),
				Strikethrough: item.Data.Done,
			})))

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

				// 	dom.Div(dom.DivProps{
				// 	Style: styles.Style{
				// 		BorderColor:   colors.PURPLE_PRIMARY,
				// 		BorderRouned:  true,
				// 		NoDefault:     true,
				// 		PaddingLeft:   styles.Int(1),
				// 		PaddingRight:  styles.Int(1),
				// 		PaddingTop:    styles.Int(1),
				// 		PaddingBottom: styles.Int(1),
				// 	},
				// 	Focusable: true,
				// 	OnKeyDown: func(d *dom.DOMEvent) {
				// 		switch d.Key {
				// 		case "up", "down":
				// 			d.PreventDefault()
				// 		case "esc":
				// 			state.SelectedEntryMode = SelectedEntryMode_Default
				// 		}
				// 	},
				// },

				// dom.Div(dom.DivProps{
				// 	Focused:   true,
				// 	Focusable: true,
				// 	OnKeyDown: func(d *dom.DOMEvent) {
				// 		switch d.Key {
				// 		case "enter":
				// 			state.OnPromote(item.ID)
				// 			state.SelectedEntryMode = SelectedEntryMode_Default

				// 			// set selected to bottom
				// 			state.SelectedEntryIndex = len(state.Entries) - 1
				// 		}
				// 	},
				// }, dom.Text("Promote", styles.Style{
				// 	BorderRouned: true,
				// 	Bold:         true,
				// })),
				// ))
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
