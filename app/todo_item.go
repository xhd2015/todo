package app

import (
	"strconv"
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

	OnNavigate func(e *dom.DOMEvent, direction int)
}

func TodoItem(props TodoItemProps) *dom.Node {
	item := props.Item
	depth := props.Depth
	isLastChild := props.IsLastChild
	isSelected := props.IsSelected
	state := props.State
	// Build tree connector prefix using common utility
	treePrefix := tree.BuildTreePrefix(depth, isLastChild)

	textColor := func() string {
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
	}()

	var textNode *dom.Node
	if state.IsSearchActive && len(item.MatchTexts) > 0 {
		nodes := make([]*dom.Node, 0, len(item.MatchTexts))
		for _, matchText := range item.MatchTexts {
			if matchText.Text == "" {
				continue
			}
			color := textColor
			if matchText.Match {
				color = colors.GREEN_SUCCESS
			}
			node := dom.Text(matchText.Text, styles.Style{
				Color:         color,
				Strikethrough: item.Data.Done,
			})
			nodes = append(nodes, node)
		}
		textNode = dom.Fragment(nodes...)
	} else {
		textNode = dom.Text(item.Data.Text, styles.Style{
			Color:         textColor,
			Strikethrough: item.Data.Done,
		})
	}

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
				if state.IsSearchActive {
					state.ClearSearch()
				}
				state.SelectedEntryMode = SelectedEntryMode_Default
			case dom.KeyTypeUp:
				if props.OnNavigate != nil {
					props.OnNavigate(e, -1)
					return
				}
			case dom.KeyTypeDown:
				if props.OnNavigate != nil {
					props.OnNavigate(e, 1)
					return
				}
			case dom.KeyTypeLeft, dom.KeyTypeRight:
				switch state.SelectedEntryMode {
				case SelectedEntryMode_DeleteConfirm:
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
				case SelectedEntryMode_Default:
					if keyEvent.KeyType == dom.KeyTypeRight {
						// show actions
						state.SelectedEntryMode = SelectedEntryMode_ShowActions
					}
				}
			case dom.KeyTypeSpace:
				// toggle status
				state.OnToggle(item.Data.ID)
			case dom.KeyTypeCtrlC:
				if state.IsSearchActive {
					state.ClearSearch()
					e.PreventDefault()
					e.StopPropagation()
				}
			default:
				key := string(keyEvent.Runes)
				switch key {
				case "/":
					// focus to input
					state.SelectedEntryID = 0
					state.Input.Focused = true
					state.LastSelectedEntryID = item.Data.ID
				case "?":
					state.SelectedEntryID = 0
					state.Input.Focused = true
					state.LastSelectedEntryID = item.Data.ID
					if !strings.HasPrefix(state.Input.Value, "?") {
						state.Input.Value = "?" + state.Input.Value
						state.Input.CursorPosition = len(state.Input.Value)
					}
				case "e":
					state.SelectedEntryMode = SelectedEntryMode_Editing
					state.SelectedInputState.FocusWithText(item.Data.Text)
				case "j":
					// move down
					if props.OnNavigate != nil {
						props.OnNavigate(e, 1)
						return
					}
				case "k":
					// move up
					if props.OnNavigate != nil {
						props.OnNavigate(e, -1)
						return
					}
				case "z":
					state.ZenMode = !state.ZenMode
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
	}, dom.Fragment(
		textNode,
		func() *dom.Node {
			if len(item.Notes) == 0 {
				return nil
			}
			return dom.Text("("+strconv.Itoa(len(item.Notes))+" notes)", styles.Style{
				Color: func() string {
					if isSelected {
						return colors.GREEN_SUCCESS
					}
					return colors.GREY_TEXT
				}(),
				Strikethrough: item.Data.Done,
			})
		}(),
	),
	)
}
