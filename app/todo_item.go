package app

import (
	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models"
)

type TodoItemProps struct {
	Item       *models.EntryView
	Index      int
	Depth      int
	IsSelected bool
	State      *State
}

func TodoItem(props TodoItemProps) *dom.Node {
	item := props.Item
	i := props.Index
	depth := props.Depth
	isSelected := props.IsSelected
	state := props.State
	indent := ""
	for d := 0; d < depth; d++ {
		indent += "  " // Two spaces per depth level
	}

	return dom.Li(dom.ListItemProps{
		Focusable: dom.Focusable(true),
		Selected:  isSelected,
		Focused:   state.SelectedEntryMode == SelectedEntryMode_Default && isSelected,
		ItemPrefix: dom.String(func() string {
			prefix := indent
			if item.Data.Done {
				prefix += "✓ "
			} else {
				prefix += "• "
			}
			return prefix
		}()),
		OnFocus: func() {
			state.SelectedEntryIndex = i
		},
		OnBlur: func() {
			state.SelectedEntryIndex = -1
		},
		OnKeyDown: func(e *dom.DOMEvent) {
			switch e.Key {
			case "/":
				// focus to input
				state.SelectedEntryIndex = -1
				state.Input.Focused = true
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
			case "a":
				// add child
				state.SelectedEntryMode = SelectedEntryMode_AddingChild
				state.ChildInputState.Value = ""
				state.ChildInputState.Focused = true
				state.ChildInputState.CursorPosition = 0
			}
		},
	}, dom.Text(item.Data.Text, styles.Style{
		Color: func() string {
			if isSelected {
				return colors.GREEN_SUCCESS
			} else if item.Data.Done {
				return ""
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
	}))
}
