package app

import (
	"context"
	"strconv"
	"strings"
	"time"

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
	// gg
	OnGoToFirst func(e *dom.DOMEvent)
	// G
	OnGoToLast func(e *dom.DOMEvent)

	// gt -> top
	OnGoToTop func(e *dom.DOMEvent)
	// gb -> bottom
	OnGoToBottom func(e *dom.DOMEvent)
}

func TodoItem(props TodoItemProps) *dom.Node {
	item := props.Item
	depth := props.Depth
	isLastChild := props.IsLastChild
	isSelected := props.IsSelected
	state := props.State
	// Build tree connector prefix using common utility
	treePrefix := tree.BuildTreePrefix(depth, isLastChild)

	inputState := &state.Input

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
			state.Select(item.Data.ID)
		},
		OnBlur: func() {
			state.Deselect()
		},
		OnKeyDown: func(e *dom.DOMEvent) {
			lastEvent := inputState.LastInputEvent
			lastTime := inputState.LastInputTime
			inputState.LastInputEvent = e
			inputState.LastInputTime = time.Now()

			keyEvent := e.KeydownEvent
			switch keyEvent.KeyType {
			case dom.KeyTypeEnter:
				if state.SelectedEntryMode == SelectedEntryMode_DeleteConfirm {
					state.SelectedEntryMode = SelectedEntryMode_Default
					return
				}
				state.Routes.Push(DetailRoute(item.Data.ID))

				item.DetailPage.InputState.Value = ""
				item.DetailPage.InputState.Focused = true
				item.DetailPage.InputState.CursorPosition = 0
			case dom.KeyTypeEsc:
				if state.IsSearchActive {
					state.ClearSearch()
				} else if state.ZenMode {
					state.ZenMode = false
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
				state.Enqueue(func(ctx context.Context) error {
					state.OnToggle(item.Data.ID)
					return nil
				})
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
					state.Deselect()
					state.Input.Focused = true
					state.LastSelectedEntryID = item.Data.ID
				case "?":
					state.Deselect()
					state.Input.Focused = true
					state.LastSelectedEntryID = item.Data.ID
					if !strings.HasPrefix(state.Input.Value, "?") {
						state.Input.Value = "?" + state.Input.Value
						state.Input.CursorPosition = len(state.Input.Value)
					}
				case "e":
					state.SelectedEntryMode = SelectedEntryMode_Editing
					state.SelectedInputState.FocusWithText(item.Data.Text)
				case "g", "t", "b":
					// Handle combinations first
					if lastEvent != nil && lastEvent.KeydownEvent != nil && time.Since(lastTime) < 5000*time.Millisecond {
						combined := string(lastEvent.KeydownEvent.Runes) + key
						switch combined {
						case "gg":
							// go to top
							// gg -> top
							if props.OnGoToFirst != nil {
								props.OnGoToFirst(e)
							}
							return
						case "gt":
							if props.OnGoToTop != nil {
								props.OnGoToTop(e)
							}
							return
						case "gb":
							if props.OnGoToBottom != nil {
								props.OnGoToBottom(e)
							}
							return
						}
					}

					// Handle standalone "t" for top command
					if key == "t" {
						if state.OnShowTop != nil {
							// Default duration is 30 minutes
							duration := 30 * time.Minute
							state.Enqueue(func(ctx context.Context) error {
								state.OnShowTop(item.Data.ID, item.Data.Text, duration)
								return nil
							})
						}
					}
				case "G":
					// go to bottom
					if props.OnGoToLast != nil {
						props.OnGoToLast(e)
					}
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
				case "x":
					// cut
					if state.CuttingEntryID == item.Data.ID {
						// Cancel cutting if pressing x on the same item
						state.CuttingEntryID = 0
					} else {
						// Start cutting this item
						state.CuttingEntryID = item.Data.ID
					}
				case "p":
					// paste
					if state.CuttingEntryID == item.Data.ID {
						// cancel
						state.CuttingEntryID = 0
						return
					}
					if state.CuttingEntryID != 0 && state.CuttingEntryID != item.Data.ID {
						// Check if the target is not a descendant of the cutting item
						if !state.IsDescendant(item.Data.ID, state.CuttingEntryID) {
							// Move the cutting item to be a child of the current item
							if state.OnMove != nil {
								state.Enqueue(func(ctx context.Context) error {
									state.OnMove(state.CuttingEntryID, item.Data.ID)
									// Clear the cutting state
									state.CuttingEntryID = 0
									return nil
								})
							}
						}
					}
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
		func() *dom.Node {
			if state.CuttingEntryID == item.Data.ID {
				return dom.Text("(cutting...)", styles.Style{
					Color: func() string {
						if isSelected {
							return colors.GREEN_SUCCESS
						}
						return colors.GREY_TEXT
					}(),
				})
			}
			return nil
		}(),
	),
	)
}
