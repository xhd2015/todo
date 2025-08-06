package app

import (
	"strings"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/ui/tree"
)

type TodoItemProps struct {
	Item        *models.LogEntryView
	Depth       int
	IsLastChild []bool
	IsSelected  bool
	State       *State
}

func TodoItem(props TodoItemProps) *dom.Node {
	item := props.Item
	depth := props.Depth
	isLastChild := props.IsLastChild
	isSelected := props.IsSelected
	state := props.State

	// Build tree connector prefix using common utility
	treePrefix := tree.BuildTreePrefix(depth, isLastChild)

	return dom.Li(dom.ListItemProps{
		Focusable: dom.Focusable(true),
		Selected:  isSelected,
		Focused:   state.SelectedEntryMode == SelectedEntryMode_Default && isSelected,
		ItemPrefix: dom.String(func() string {
			prefix := treePrefix
			if item.Data.Done {
				prefix += "✓"
			} else {
				prefix += "•"
			}
			return prefix
		}()),
		OnFocus: func() {
			state.SelectedEntryID = item.Data.ID
		},
		OnBlur: func() {
			state.SelectedEntryID = 0
		},
		OnKeyDown: func(e *dom.DOMEvent) {
			keyEvent := e.KeydownEvent
			if keyEvent == nil {
				return
			}
			switch keyEvent.KeyType {
			case dom.KeyTypeEnter:
				if state.SelectedEntryMode == SelectedEntryMode_DeleteConfirm {
					state.SelectedEntryMode = SelectedEntryMode_Default
					return
				}
				state.EnteredEntryID = item.Data.ID

				item.DetailPage.InputState.Value = ""
				item.DetailPage.InputState.Focused = true
				item.DetailPage.InputState.CursorPosition = 0
			case dom.KeyTypeEsc:
				state.SelectedEntryMode = SelectedEntryMode_Default
			case dom.KeyTypeUp, dom.KeyTypeDown:
				state.SelectedEntryMode = SelectedEntryMode_Default
			case dom.KeyTypeLeft, dom.KeyTypeRight:
				if state.SelectedEntryMode == SelectedEntryMode_DeleteConfirm {
					delta := 1
					if keyEvent.KeyType == dom.KeyTypeLeft {
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
					if keyEvent.KeyType == dom.KeyTypeRight {
						// show actions
						state.SelectedEntryMode = SelectedEntryMode_ShowActions
					}
				}
			case dom.KeyTypeSpace:
				// toggle status
				state.OnToggle(item.Data.ID)
			default:
				key := string(keyEvent.Runes)
				switch key {
				case "/":
					// focus to input
					state.SelectedEntryID = 0
					state.Input.Focused = true
				case "?":
					state.SelectedEntryID = 0
					state.Input.Focused = true
					if !strings.HasPrefix(state.Input.Value, "?") {
						state.Input.Value = "?" + state.Input.Value
						state.Input.CursorPosition = len(state.Input.Value)
					}
				case "e":
					state.SelectedEntryMode = SelectedEntryMode_Editing
					state.SelectedInputState.Value = item.Data.Text
					state.SelectedInputState.Focused = true
					state.SelectedInputState.CursorPosition = len(item.Data.Text) + 1
				case "j":
					// move down
					next := state.Entries.FindNextOrLast(state.SelectedEntryID)
					var nextID int64
					if next != nil {
						nextID = next.Data.ID
					}
					state.SelectedEntryID = nextID
				case "k":
					// move up
					prev := state.Entries.FindPrevOrFirst(state.SelectedEntryID)
					var prevID int64
					if prev != nil {
						prevID = prev.Data.ID
					}
					state.SelectedEntryID = prevID
				case "d":
					state.SelectedEntryMode = SelectedEntryMode_DeleteConfirm
					state.SelectedDeleteConfirmButton = 0
				case "a":
					// add child
					state.SelectedEntryMode = SelectedEntryMode_AddingChild
					state.ChildInputState.Value = ""
					state.ChildInputState.Focused = true
					state.ChildInputState.CursorPosition = 0
				}
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
