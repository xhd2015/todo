package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models"
)

// TreeLog represents a flattened log entry
type TreeLog struct {
	Entry *models.LogEntryView
}

// TreeNote represents a flattened note
type TreeNote struct {
	Note    *models.NoteView
	EntryID int64 // ID of the entry that owns this note
}

// TreeEntryType represents the type of entry in a TreeEntry
type TreeEntryType string

const (
	TreeEntryType_Log  TreeEntryType = "Log"
	TreeEntryType_Note TreeEntryType = "Note"
)

// TreeEntry wraps either a log entry or a note for unified tree rendering
type TreeEntry struct {
	Type   TreeEntryType
	Prefix string
	IsLast bool

	Log  *TreeLog
	Note *TreeNote
}

func (c *TreeEntry) Text() string {
	switch c.Type {
	case TreeEntryType_Log:
		return c.Log.Entry.Data.Text
	case TreeEntryType_Note:
		return c.Note.Note.Data.Text
	}
	return ""
}

// TreeProps contains configuration for rendering the entry tree
type TreeProps struct {
	State        *State // The application state
	Entries      []TreeEntry
	EntriesAbove int
	EntriesBelow int

	OnNavigate   func(e *dom.DOMEvent, entryType TreeEntryType, entryID int64, direction int)
	OnEnter      func(e *dom.DOMEvent, entryType TreeEntryType, entryID int64)
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
	selectedID := state.SelectedEntryID
	if selectedID != 0 && len(entries) > 0 {
		var hasSelected bool
		for _, wrapperEntry := range entries {
			if wrapperEntry.Type == TreeEntryType_Log && wrapperEntry.Log != nil && wrapperEntry.Log.Entry.Data.ID == selectedID {
				hasSelected = true
				break
			}
		}
		if !hasSelected && len(entries) > 0 && entries[0].Type == TreeEntryType_Log && entries[0].Log != nil {
			selectedID = entries[0].Log.Entry.Data.ID
		}
	}

	for _, entry := range entries {
		if entry.Type == TreeEntryType_Log && entry.Log != nil {
			logEntry := entry.Log
			item := logEntry.Entry
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
							state.Enqueue(func(ctx context.Context) error {
								return state.OnUpdate(item.Data.ID, state.SelectedInputState.Value)
							})
							state.SelectedEntryMode = SelectedEntryMode_Default
						}
					},
				}))
				continue
			}

			// Always render the TodoItem
			children = append(children, TodoItem(TodoItemProps{
				Item:       item,
				Prefix:     entry.Prefix,
				IsLast:     entry.IsLast,
				IsSelected: isSelected,
				State:      state,
				OnNavigate: func(e *dom.DOMEvent, direction int) {
					if props.OnNavigate != nil {
						props.OnNavigate(e, TreeEntryType_Log, item.Data.ID, direction)
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
						props.OnEnter(e, TreeEntryType_Log, entryID)
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
									id, err := state.OnAddChild(item.Data.ID, state.ChildInputState.Value)
									if err != nil {
										return err
									}

									state.ChildInputState.Value = ""
									state.ChildInputState.CursorPosition = 0
									state.Select(id)
									return nil
								})
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
						state.Enqueue(func(ctx context.Context) error {
							err := state.OnDelete(item.Data.ID)
							if err != nil {
								return err
							}
							var nextID int64
							if next != nil {
								nextID = next.Data.ID
							}
							state.Select(nextID)
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
								err := state.OnPromote(item.Data.ID)
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
		} else if entry.Type == TreeEntryType_Note && entry.Note != nil {
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
						props.OnNavigate(e, TreeEntryType_Note, note.Data.ID, direction)
					}
				},
				OnEnter: func(e *dom.DOMEvent, entryID int64) {
					if props.OnEnter != nil {
						props.OnEnter(e, TreeEntryType_Note, entryID)
					}
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
