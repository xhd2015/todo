package app

import (
	"fmt"
	"strings"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models"
)

// EntryWithDepth represents a flattened entry with its depth and ancestor information
type EntryWithDepth struct {
	Entry       *models.LogEntryView
	Depth       int
	IsLastChild []bool // For each depth level, whether this entry is the last child at that level
}

// RenderEntryTreeProps contains configuration for rendering the entry tree
type RenderEntryTreeProps struct {
	State        *State // The application state
	Entries      []EntryWithDepth
	EntriesAbove int
	EntriesBelow int

	OnNavigate   func(e *dom.DOMEvent, entryID int64, direction int)
	OnGoToFirst  func(e *dom.DOMEvent)
	OnGoToLast   func(e *dom.DOMEvent)
	OnGoToTop    func(e *dom.DOMEvent)
	OnGoToBottom func(e *dom.DOMEvent)
}

// RenderEntryTree builds and renders the tree of entries as DOM nodes
func RenderEntryTree(props RenderEntryTreeProps) []*dom.Node {
	state := props.State

	entriesAbove := props.EntriesAbove
	entriesBelow := props.EntriesBelow
	entries := props.Entries

	// Apply pagination to flatEntries
	showUpIndicator := entriesAbove > 0
	showDownIndicator := entriesBelow > 0

	var children []*dom.Node

	// Add up indicator if needed
	if showUpIndicator {
		message := fmt.Sprintf("↑ %d more above", entriesAbove)
		children = append(children, dom.Text(message, styles.Style{
			Color: colors.GREY_TEXT,
		}),
			dom.Br(),
		)
	}

	// auto select first if selected one is hidden
	selectedID := state.SelectedEntryID
	if selectedID != 0 && len(entries) > 0 {
		var hasSelected bool
		for _, entryWithDepth := range entries {
			if entryWithDepth.Entry.Data.ID == selectedID {
				hasSelected = true
				break
			}
		}
		if !hasSelected {
			selectedID = entries[0].Entry.Data.ID
		}
	}

	for _, entryWithDepth := range entries {
		item := entryWithDepth.Entry
		depth := entryWithDepth.Depth
		isSelected := selectedID == item.Data.ID

		if state.SelectedEntryMode == SelectedEntryMode_Editing && isSelected {
			children = append(children, dom.Input(dom.InputProps{
				Value:          state.SelectedInputState.Value,
				Focused:        state.SelectedInputState.Focused,
				CursorPosition: state.SelectedInputState.CursorPosition,
				OnCursorMove: func(position int) {
					state.SelectedInputState.CursorPosition = position
				},
				OnChange: func(value string) {
					state.SelectedInputState.Value = value
				},
				OnKeyDown: func(e *dom.DOMEvent) {
					keyEvent := e.KeydownEvent
					switch keyEvent.KeyType {
					case dom.KeyTypeUp, dom.KeyTypeDown:
						e.PreventDefault()
					case dom.KeyTypeEsc:
						state.SelectedEntryMode = SelectedEntryMode_Default
					case dom.KeyTypeCtrlC:
						state.SelectedEntryMode = SelectedEntryMode_Default
						e.StopPropagation()
					case dom.KeyTypeEnter:
						state.OnUpdate(item.Data.ID, state.SelectedInputState.Value)
						state.SelectedEntryMode = SelectedEntryMode_Default
					}
				},
			}))
			continue
		}

		// Always render the TodoItem
		children = append(children, TodoItem(TodoItemProps{
			Item:        item,
			Depth:       depth,
			IsLastChild: entryWithDepth.IsLastChild,
			IsSelected:  isSelected,
			State:       state,
			OnNavigate: func(e *dom.DOMEvent, direction int) {
				if props.OnNavigate != nil {
					props.OnNavigate(e, item.Data.ID, direction)
				}
			},
			OnGoToFirst: func(e *dom.DOMEvent) {
				if props.OnGoToFirst != nil {
					props.OnGoToFirst(e)
				}
			},
			OnGoToLast: func(e *dom.DOMEvent) {
				if props.OnGoToLast != nil {
					props.OnGoToLast(e)
				}
			},
			OnGoToTop: func(e *dom.DOMEvent) {
				if props.OnGoToTop != nil {
					props.OnGoToTop(e)
				}
			},
			OnGoToBottom: func(e *dom.DOMEvent) {
				if props.OnGoToBottom != nil {
					props.OnGoToBottom(e)
				}
			},
		}))

		// Render child input box under the item when in AddingChild mode
		if state.SelectedEntryMode == SelectedEntryMode_AddingChild && isSelected {
			children = append(children, dom.Input(dom.InputProps{
				Placeholder:    "add breakdown",
				Value:          state.ChildInputState.Value,
				Focused:        state.ChildInputState.Focused,
				CursorPosition: state.ChildInputState.CursorPosition,
				OnCursorMove: func(position int) {
					state.ChildInputState.CursorPosition = position
				},
				OnChange: func(value string) {
					state.ChildInputState.Value = value
				},
				OnKeyDown: func(e *dom.DOMEvent) {
					keyEvent := e.KeydownEvent
					switch keyEvent.KeyType {
					case dom.KeyTypeUp, dom.KeyTypeDown:
						e.PreventDefault()
					case dom.KeyTypeEsc:
						state.SelectedEntryMode = SelectedEntryMode_Default
					case dom.KeyTypeEnter:
						if strings.TrimSpace(state.ChildInputState.Value) != "" {
							id, err := state.OnAddChild(item.Data.ID, state.ChildInputState.Value)
							if err != nil {
								// TODO: show error
								panic(err)
								return
							}
							state.ChildInputState.Value = ""
							state.ChildInputState.CursorPosition = 0
							state.Select(id)
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
					next := state.Entries.FindNextOrLast(item.Data.ID)
					state.OnDelete(item.Data.ID)
					var nextID int64
					if next != nil {
						nextID = next.Data.ID
					}
					state.Select(nextID)
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
			HIGHLIGHTS := 5
			items := []MenuItem{
				{Text: "Promote", OnSelect: func() {
					state.OnPromote(item.Data.ID)
					state.SelectedEntryMode = SelectedEntryMode_Default
				}},
				{Text: "No Highlight", OnSelect: func() {
					state.OnUpdateHighlight(item.Data.ID, 0)
					state.SelectedEntryMode = SelectedEntryMode_Default
				}},
			}
			colors := []string{
				colors.DARK_RED_1,
				colors.DARK_RED_2,
				colors.DARK_RED_3,
				colors.DARK_RED_4,
				colors.DARK_RED_5,
			}
			for i := 0; i < HIGHLIGHTS; i++ {
				items = append(items, MenuItem{
					Text:  fmt.Sprintf("Highlight-%d", i+1),
					Color: colors[i],
					OnSelect: func() {
						state.OnUpdateHighlight(item.Data.ID, i+1)
						state.SelectedEntryMode = SelectedEntryMode_Default
					},
				})
			}

			children = append(children, Menu(MenuProps{
				Title:         "Promote",
				SelectedIndex: state.SelectedActionIndex,
				OnSelect: func(index int) {
					state.SelectedActionIndex = index
				},
				Items: items,
				OnKeyDown: func(e *dom.DOMEvent) {
					keyEvent := e.KeydownEvent
					switch keyEvent.KeyType {
					case dom.KeyTypeUp, dom.KeyTypeDown:
						e.PreventDefault()
					}
				},
				OnDismiss: func() {
					state.SelectedEntryMode = SelectedEntryMode_Default
				},
			}))
		}
	}

	// Add down indicator if needed
	if showDownIndicator {
		message := fmt.Sprintf("↓ %d more below", entriesBelow)
		children = append(children, dom.Text(message, styles.Style{
			Color: colors.GREY_TEXT,
		}),
			dom.Br(),
		)
	}

	return children
}

func getLines(SelectedEntryMode SelectedEntryMode) int {
	switch SelectedEntryMode {
	case SelectedEntryMode_Default:
		return 1
	case SelectedEntryMode_AddingChild:
		return 2
	case SelectedEntryMode_Editing:
		return 2
	case SelectedEntryMode_DeleteConfirm:
		return 2
	case SelectedEntryMode_ShowActions:
		return 9
	default:
		return 1
	}
}
