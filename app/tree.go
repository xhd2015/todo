package app

import (
	"fmt"
	"strings"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/ui/search"
)

// EntryWithDepth represents a flattened entry with its depth and ancestor information
type EntryWithDepth struct {
	Entry       *models.LogEntryView
	Depth       int
	IsLastChild []bool // For each depth level, whether this entry is the last child at that level
}

// RenderEntryTree builds and renders the tree of entries as DOM nodes
func RenderEntryTree(state *State) []*dom.Node {
	var flatEntries []EntryWithDepth

	var addEntryRecursive func(entry *models.LogEntryView, depth int, ancestorIsLast []bool)
	addEntryRecursive = func(entry *models.LogEntryView, depth int, ancestorIsLast []bool) {
		flatEntries = append(flatEntries, EntryWithDepth{
			Entry:       entry,
			Depth:       depth,
			IsLastChild: ancestorIsLast,
		})

		// Add children recursively
		for childIndex, child := range entry.Children {
			isLastChild := (childIndex == len(entry.Children)-1)
			// Create ancestor info for child: copy parent's info and add current level
			childAncestorIsLast := make([]bool, depth+1)
			copy(childAncestorIsLast, ancestorIsLast)
			childAncestorIsLast[depth] = isLastChild
			addEntryRecursive(child, depth+1, childAncestorIsLast)
		}
	}

	// Filter entries based on search query if active
	entriesToRender := state.Entries
	if state.IsSearchActive && state.SearchQuery != "" {
		entriesToRender = search.FilterEntriesRecursive(state.Entries, state.SearchQuery)
	}

	// Add top-level entries (ParentID == 0)
	topLevelEntries := make([]*models.LogEntryView, 0)
	for _, entry := range entriesToRender {
		if entry.Data.ParentID == 0 {
			topLevelEntries = append(topLevelEntries, entry)
		}
	}

	for _, entry := range topLevelEntries {
		addEntryRecursive(entry, 0, []bool{})
	}

	var children []*dom.Node
	for _, entryWithDepth := range flatEntries {
		item := entryWithDepth.Entry
		depth := entryWithDepth.Depth
		isSelected := state.SelectedEntryID == item.Data.ID

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
			Item:        item,
			Depth:       depth,
			IsLastChild: entryWithDepth.IsLastChild,
			IsSelected:  isSelected,
			State:       state,
		}))

		// Render child input box under the item when in AddingChild mode
		if state.SelectedEntryMode == SelectedEntryMode_AddingChild && isSelected {
			children = append(children, dom.Input(dom.InputProps{
				Placeholder:    "add breakdown",
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
					next := state.Entries.FindNextOrLast(item.Data.ID)
					state.OnDelete(item.Data.ID)
					var nextID int64
					if next != nil {
						nextID = next.Data.ID
					}
					state.SelectedEntryID = nextID
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

	return children
}
