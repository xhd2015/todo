package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/app/emojis"
	"github.com/xhd2015/todo/models"
)

// TreeLog represents a flattened log entry
type TreeLog struct {
}

// TreeNote represents a flattened note
type TreeNote struct {
	Note    *models.NoteView
	EntryID int64 // ID of the entry that owns this note
}

// TreeFocusedItem represents the focused root path
type TreeFocusedItem struct {
	RootPath []string
}

// TreeGroup represents a group entry
type TreeGroup struct {
	ID   int64
	Name string
}

// TreeEntry wraps either a log entry or a note for unified tree rendering
type TreeEntry struct {
	Type   models.LogEntryViewType
	Prefix string
	IsLast bool

	// for all
	Entry *models.LogEntryView

	Log         *TreeLog
	Note        *TreeNote
	FocusedItem *TreeFocusedItem
	Group       *TreeGroup
}

func (c *TreeEntry) Text() string {
	switch c.Type {
	case models.LogEntryViewType_Log:
		if c.Entry != nil && c.Entry.Data != nil {
			return c.Entry.Data.Text
		}
		return ""
	case models.LogEntryViewType_Note:
		return c.Note.Note.Data.Text
	case models.LogEntryViewType_FocusedItem:
		if c.FocusedItem != nil && len(c.FocusedItem.RootPath) > 0 {
			// Join the path components with " > " separator
			result := c.FocusedItem.RootPath[0]
			for i := 1; i < len(c.FocusedItem.RootPath); i++ {
				result += " > " + c.FocusedItem.RootPath[i]
			}
			return result
		}
		return ""
	case models.LogEntryViewType_Group:
		if c.Group != nil {
			return c.Group.Name
		}
		return ""
	}
	return ""
}

// TreeProps contains configuration for rendering the entry tree
type TreeProps struct {
	State        *State // The application state
	Entries      []TreeEntry
	EntriesAbove int
	EntriesBelow int

	OnNavigate   func(e *dom.DOMEvent, entryType models.LogEntryViewType, entryID int64, direction int)
	OnEnter      func(e *dom.DOMEvent, entryType models.LogEntryViewType, entryID int64)
	OnGoToFirst  func(e *dom.DOMEvent)
	OnGoToLast   func(e *dom.DOMEvent)
	OnGoToTop    func(e *dom.DOMEvent)
	OnGoToBottom func(e *dom.DOMEvent)
}

// Tree builds and renders the tree of entries as DOM nodes
func Tree(props TreeProps) []*dom.Node {
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
	effectiveSelectedEntry := state.SelectedEntry
	if state.SelectedEntry.IsSet() && len(entries) > 0 {
		var hasSelected bool
		for _, wrapperEntry := range entries {
			if wrapperEntry.Entry != nil && wrapperEntry.Entry.SameIdentity(state.SelectedEntry) {
				hasSelected = true
				break
			}
		}
		if !hasSelected {
			// Find first entry with non-nil Entry field
			for _, entry := range entries {
				if entry.Entry != nil {
					effectiveSelectedEntry = entry.Entry.Identity()
					break
				}
			}
		}
	}

	for _, entry := range entries {
		var entryItem *models.LogEntryView
		var entryIdentity models.EntryIdentity

		if entry.Type == models.LogEntryViewType_Log || entry.Type == models.LogEntryViewType_Group {
			entryItem = entry.Entry
			if entry.Entry != nil {
				entryIdentity = entry.Entry.Identity()
			}
		}

		if entryItem != nil {
			isSelected := effectiveSelectedEntry == entryIdentity
			entryType := entryIdentity.EntryType
			entryID := entryIdentity.ID

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
							state.Enqueue(func(ctx context.Context) error {
								return state.OnUpdate(entryType, entryID, state.SelectedInputState.Value)
							})
							state.SelectedEntryMode = SelectedEntryMode_Default
						}
					},
				}))
				continue
			}

			// Always render the TodoItem
			children = append(children, TodoItem(TodoItemProps{
				Item:       entryItem,
				Prefix:     entry.Prefix,
				IsLast:     entry.IsLast,
				IsSelected: isSelected,
				State:      state,
				OnNavigate: func(e *dom.DOMEvent, direction int) {
					if props.OnNavigate != nil {
						props.OnNavigate(e, entryType, entryID, direction)
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
				OnEnter: func(e *dom.DOMEvent, entryID int64) {
					if props.OnEnter != nil {
						props.OnEnter(e, entryType, entryID)
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
								state.Enqueue(func(ctx context.Context) error {
									id, err := state.OnAddChild(entryType, entryID, state.ChildInputState.Value)
									if err != nil {
										return err
									}

									state.ChildInputState.Value = ""
									state.ChildInputState.CursorPosition = 0
									state.Select(entryType, id)
									return nil
								})
							}
							state.SelectedEntryMode = SelectedEntryMode_Default
						}
					},
				}))
			}

			if state.SelectedEntryMode == SelectedEntryMode_DeleteConfirm && isSelected {
				deleteText := "Delete todo?"
				deleteButtonText := "[Delete]"
				if props.State.ViewMode == ViewMode_Group {
					deleteText = "Remove from group?"
					deleteButtonText = "[Remove]"
				}

				children = append(children, ConfirmDialog(ConfirmDialogProps{
					SelectedButton: state.SelectedDeleteConfirmButton,
					PromptText:     deleteText,
					DeleteText:     deleteButtonText,
					CancelText:     "[Cancel]",
					OnDelete: func() {
						next := state.Entries.FindNextOrLast(entryID)
						state.Enqueue(func(ctx context.Context) error {
							if props.State.ViewMode == ViewMode_Group {
								err := state.OnRemoveFromGroup(entryType, entryID)
								if err != nil {
									return err
								}
							} else {
								err := state.OnDelete(entryType, entryID)
								if err != nil {
									return err
								}
							}
							var nextID int64
							if next != nil {
								nextID = next.Data.ID
							}
							state.Select(entryType, nextID)
							state.SelectedEntryMode = SelectedEntryMode_Default
							return nil
						})
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
					{
						Text: "Promote",
						OnSelect: func() {
							state.Enqueue(func(ctx context.Context) error {
								err := state.OnPromote(entryType, entryID)
								if err != nil {
									return err
								}
								state.SelectedEntryMode = SelectedEntryMode_Default
								return nil
							})
						}},
					{
						Text: "No Highlight",
						OnSelect: func() {
							state.OnUpdateHighlight(entryType, entryID, 0)
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
							state.OnUpdateHighlight(entryType, entryID, i+1)
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
		} else if entry.Type == models.LogEntryViewType_Note && entry.Note != nil {
			// Handle note rendering
			noteEntry := entry.Note
			note := noteEntry.Note
			isSelected := state.SelectedNoteID == note.Data.ID

			children = append(children, TodoNote(TodoNoteProps{
				Note:       note,
				EntryID:    noteEntry.EntryID,
				Prefix:     entry.Prefix,
				IsLast:     entry.IsLast,
				IsSelected: isSelected,
				State:      state,
				OnNavigate: func(e *dom.DOMEvent, direction int) {
					if props.OnNavigate != nil {
						props.OnNavigate(e, models.LogEntryViewType_Note, note.Data.ID, direction)
					}
				},
				OnEnter: func(e *dom.DOMEvent, entryID int64) {
					if props.OnEnter != nil {
						props.OnEnter(e, models.LogEntryViewType_Note, entryID)
					}
				},
			}))
		} else if entry.Type == models.LogEntryViewType_FocusedItem && entry.FocusedItem != nil {
			// Handle focused root path rendering
			children = append(children, dom.Li(dom.ListItemProps{
				Focusable:  dom.Focusable(false),
				Selected:   false, // Focused items are not selectable like regular entries
				Focused:    false,
				ItemPrefix: dom.String(emojis.FOCUSED + " "), // Use a location icon to distinguish it
			}, dom.Text(entry.Text(), styles.Style{
				Color: colors.PURPLE_PRIMARY,
			})))
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
