package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/app/emojis"
	"github.com/xhd2015/todo/log"
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/ui/tree"
)

type TodoItemProps struct {
	Item       *models.LogEntryView
	Prefix     string
	IsLast     bool
	IsSelected bool
	State      *State

	OnNavigate func(e *dom.DOMEvent, direction int)
	OnEnter    func(e *dom.DOMEvent, entryID int64)
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
	entryIdentiy := item.Identity()
	entryType := entryIdentiy.EntryType
	entryID := entryIdentiy.ID

	prefix := props.Prefix
	isLast := props.IsLast
	isSelected := props.IsSelected
	state := props.State
	// Build tree connector prefix using common utility
	treePrefix := tree.BuildTreePrefix(prefix, isLast)

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
			listIndicator := emojis.LIST_DOT
			if entryType == models.LogEntryViewType_Group {
				listIndicator = emojis.FOLDER
			} else if item.Data.Done {
				listIndicator = emojis.CHECKED
			}
			return treePrefix + listIndicator
		}()),
		OnFocus: func() {
			log.Infof(context.TODO(), "focused: %v, %v", entryType, entryID)
			state.Select(entryType, entryID)
		},
		OnBlur: func() {
			log.Infof(context.TODO(), "blurred: %v, %v", entryType, entryID)
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
				// If search mode is active, clear search instead of entering detail page
				if state.IsSearchActive {
					state.ClearSearch()
					return
				}
				if props.OnEnter != nil {
					props.OnEnter(e, entryID)
				}
			case dom.KeyTypeEsc:
				if state.IsSearchActive {
					state.ClearSearch()
				} else if state.FocusedEntry.IsSet() {
					state.FocusedEntry.Unset()
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
					return state.OnToggle(entryType, entryID)
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
					state.LastSelectedEntry = models.EntryIdentity{
						EntryType: entryType,
						ID:        entryID,
					}
				case "?":
					state.Deselect()
					state.Input.Focused = true
					state.LastSelectedEntry = models.EntryIdentity{
						EntryType: entryType,
						ID:        entryID,
					}
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
								state.OnShowTop(entryID, item.Data.Text, duration)
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
				case "l":
					// Navigate to parent and push current to stack (only in group mode)
					if state.ViewMode == ViewMode_Group {
						// Push current entry to navigation stack
						state.PushToNavigationStack(entryIdentiy)

						// Get current entry and navigate to parent
						currentEntry := state.FindEntryByID(entryID)
						if currentEntry != nil {
							parentID := currentEntry.Data.ParentID
							if parentID == 0 {
								// No parent, focus on the group this entry belongs to
								groupID := state.findGroupForEntry(entryID)
								if groupID > 0 {
									state.Select(models.LogEntryViewType_Group, groupID)
								}
							} else {
								// Navigate to parent
								state.Select(models.LogEntryViewType_Log, parentID)
							}
						}
					}
				case "L":
					// Pop from navigation stack and focus previous entry (only in group mode)
					if state.ViewMode == ViewMode_Group {
						if prevEntry, found := state.PopFromNavigationStack(); found {
							state.Select(prevEntry.EntryType, prevEntry.ID)
						}
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
					if state.CuttingEntry == entryIdentiy {
						// Cancel cutting if pressing x on the same item
						state.CuttingEntry.Unset()
					} else {
						// Start cutting this item
						state.CuttingEntry = entryIdentiy
					}
				case "p":
					// paste
					if state.CuttingEntry == entryIdentiy {
						// cancel
						state.CuttingEntry.Unset()
						return
					}
					if state.CuttingEntry.IsSet() && state.CuttingEntry != entryIdentiy {
						// Check if the target is not a descendant of the cutting item
						if !state.IsDescendantOf(entryIdentiy, state.CuttingEntry) {
							// Move the cutting item to be a child of the current item
							if state.OnMove != nil {
								state.Enqueue(func(ctx context.Context) error {
									err := state.OnMove(state.CuttingEntry, entryIdentiy)
									if err != nil {
										return err
									}
									// Clear the cutting state
									state.CuttingEntry.Unset()
									return nil
								})
							}
						}
					}
				case "v":
					// toggle history inclusion for children (also enables notes)
					if state.OnToggleVisibility != nil {
						state.Enqueue(func(ctx context.Context) error {
							err := state.OnToggleVisibility(entryID)
							if err != nil {
								return err
							}
							return nil
						})
					}
				case "n":
					// toggle notes display for this entry and its subtree
					if state.OnToggleNotesDisplay != nil {
						state.Enqueue(func(ctx context.Context) error {
							err := state.OnToggleNotesDisplay(entryID)
							if err != nil {
								return err
							}
							return nil
						})
					}
				case "f":
					// enter focused mode on this entry
					state.FocusedEntry = models.EntryIdentity{
						EntryType: entryType,
						ID:        entryID,
					}
				case ",":
					// toggle collapsed state
					if state.OnToggleCollapsed != nil {
						state.Enqueue(func(ctx context.Context) error {
							err := state.OnToggleCollapsed(entryType, entryID)
							if err != nil {
								return err
							}
							return nil
						})
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
			if item.IncludeHistory {
				return dom.Text(" (*)", styles.Style{
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
		func() *dom.Node {
			// Only show collapsed indicator if the entry is collapsed AND expandAll is not active
			if item.Data.Collapsed && !props.State.ExpandAll {
				var text string
				if item.CollapsedCount > 0 {
					text = fmt.Sprintf(" (%d collapsed)", item.CollapsedCount)
				} else {
					text = " (collapsed)"
				}
				return dom.Text(text, styles.Style{
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
		func() *dom.Node {
			if state.CuttingEntry == entryIdentiy {
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

type TodoNoteProps struct {
	Note       *models.NoteView
	EntryID    int64 // ID of the entry that owns this note
	Prefix     string
	IsLast     bool
	IsSelected bool
	State      *State

	OnNavigate func(e *dom.DOMEvent, direction int)
	OnEnter    func(e *dom.DOMEvent, entryID int64)
}

func TodoNote(props TodoNoteProps) *dom.Node {
	note := props.Note
	prefix := props.Prefix
	isLast := props.IsLast
	isSelected := props.IsSelected

	// Build tree connector prefix using common utility
	treePrefix := tree.BuildTreePrefix(prefix, isLast)

	textColor := func() string {
		if isSelected {
			return colors.GREEN_SUCCESS
		}
		return colors.GREY_TEXT
	}()

	// Create text node with highlighting support
	var noteTextNode *dom.Node
	if props.State.IsSearchActive && len(note.MatchTexts) > 0 {
		nodes := make([]*dom.Node, 0, len(note.MatchTexts))
		for _, matchText := range note.MatchTexts {
			if matchText.Text == "" {
				continue
			}
			color := textColor
			if matchText.Match {
				color = colors.GREEN_SUCCESS
			}
			node := dom.Text(matchText.Text, styles.Style{
				Color: color,
			})
			nodes = append(nodes, node)
		}
		noteTextNode = dom.Fragment(nodes...)
	} else {
		noteTextNode = dom.Text(note.Data.Text, styles.Style{
			Color: textColor,
		})
	}

	return dom.Li(dom.ListItemProps{
		Focusable: dom.Focusable(true),
		Selected:  isSelected,
		Focused:   isSelected,
		ItemPrefix: dom.String(func() string {
			prefix := treePrefix
			prefix += "📝 " // Note icon with extra space
			return prefix
		}()),
		OnFocus: func() {
			props.State.SelectNote(note.Data.ID, props.EntryID)
		},
		OnBlur: func() {
			props.State.Deselect()
		},
		OnKeyDown: func(e *dom.DOMEvent) {
			keyEvent := e.KeydownEvent
			switch keyEvent.KeyType {
			case dom.KeyTypeEnter:
				if props.OnEnter != nil {
					props.OnEnter(e, props.EntryID)
				}
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
			default:
				key := string(keyEvent.Runes)
				switch key {
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
				}
			}
		},
	}, noteTextNode)
}
