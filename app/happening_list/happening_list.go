package happening_list

import (
	"strings"
	"time"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models"
)

// HappeningListProps represents the props for the HappeningList component
type HappeningListProps struct {
	Items          []*models.Happening
	InputState     *models.InputState
	OnNavigateBack func()
	OnAddHappening func(text string)
	OnReload       func() // New callback for reloading the list

	FocusedItemID int64
	OnFocusItem   func(id int64)
	OnBlurItem    func(id int64)

	// Edit/Delete functionality
	EditingItemID           int64
	EditInputState          *models.InputState
	DeletingItemID          int64
	DeleteConfirmButton     int
	OnEditItem              func(id int64)
	OnDeleteItem            func(id int64)
	OnSaveEdit              func(id int64, content string)
	OnCancelEdit            func(e *dom.DOMEvent)
	OnConfirmDelete         func(e *dom.DOMEvent, id int64)
	OnCancelDelete          func(e *dom.DOMEvent)
	OnNavigateDeleteConfirm func(direction int)
}

// HappeningList renders a list of happening items
func HappeningList(props HappeningListProps) *dom.Node {
	// Create children nodes for each happening item
	children := make([]*dom.Node, 0, len(props.Items))

	// Add each happening item
	for _, item := range props.Items {
		itemID := item.ID // Capture item ID for closure
		children = append(children, HappeningItem(&HappeningItemProps{
			Item:    item,
			Focused: props.FocusedItemID == item.ID,
			OnFocus: func() {
				props.OnFocusItem(itemID)
			},
			OnBlur: func() {
				props.OnBlurItem(itemID)
			},
			// Edit/Delete functionality
			IsEditing:           props.EditingItemID == item.ID,
			IsDeleting:          props.DeletingItemID == item.ID,
			EditInputState:      props.EditInputState,
			DeleteConfirmButton: props.DeleteConfirmButton,
			OnEdit: func() {
				if props.OnEditItem != nil {
					props.OnEditItem(itemID)
				}
			},
			OnDelete: func() {
				if props.OnDeleteItem != nil {
					props.OnDeleteItem(itemID)
				}
			},
			OnSaveEdit: func(content string) {
				if props.OnSaveEdit != nil {
					props.OnSaveEdit(itemID, content)
				}
			},
			OnCancelEdit: func(e *dom.DOMEvent) {
				if props.OnCancelEdit != nil {
					props.OnCancelEdit(e)
				}
			},
			OnConfirmDelete: func(e *dom.DOMEvent) {
				if props.OnConfirmDelete != nil {
					props.OnConfirmDelete(e, itemID)
				}
			},
			OnCancelDelete: func(e *dom.DOMEvent) {
				if props.OnCancelDelete != nil {
					props.OnCancelDelete(e)
				}
			},
			OnNavigateDeleteConfirm: func(direction int) {
				if props.OnNavigateDeleteConfirm != nil {
					props.OnNavigateDeleteConfirm(direction)
				}
			},
			// Key event handling - moved from HappeningItem
			OnKeyEvent: func(e *dom.DOMEvent) {
				handleItemKeyEvent(e, itemID, &props)
			},
		}))
	}

	// If no items, show empty message
	if len(props.Items) == 0 {
		children = append(children,
			dom.P(
				dom.DivProps{},
				dom.Text("No happenings yet.", styles.Style{
					Color: "#888888",
				}),
			),
		)
	}

	// Add input box at the end
	if props.InputState != nil {
		children = append(children,
			dom.Br(), // Add some spacing
			dom.Input(dom.InputProps{
				Placeholder:    "add happening or /todo to go back",
				Value:          props.InputState.Value,
				Focused:        props.InputState.Focused,
				CursorPosition: props.InputState.CursorPosition,
				Focusable:      dom.Focusable(true),
				Width:          50,
				OnFocus: func() {
					props.InputState.Focused = true
				},
				OnBlur: func() {
					props.InputState.Focused = false
				},
				OnChange: func(value string) {
					props.InputState.Value = value
				},
				OnCursorMove: func(position int) {
					if position < 0 {
						position = 0
					}
					valueLen := len([]rune(props.InputState.Value))
					if position > valueLen {
						position = valueLen
					}
					props.InputState.CursorPosition = position
				},
				OnKeyDown: func(event *dom.DOMEvent) {
					keyEvent := event.KeydownEvent
					if keyEvent.KeyType == dom.KeyTypeEnter {
						text := strings.TrimSpace(props.InputState.Value)
						if text == "" {
							return
						}

						// Clear input
						props.InputState.Value = ""
						props.InputState.CursorPosition = 0

						// Handle commands
						switch text {
						case "/todo":
							if props.OnNavigateBack != nil {
								props.OnNavigateBack()
							}
							return
						case "/reload", "/refresh":
							if props.OnReload != nil {
								props.OnReload()
							}
							return
						default:
							// Handle other text as new happening
							if props.OnAddHappening != nil {
								props.OnAddHappening(text)
							}
						}
					}
				},
			}),
		)
	}

	return dom.Div(
		dom.DivProps{},
		children...,
	)
}

// handleItemKeyEvent handles key events for happening items
func handleItemKeyEvent(e *dom.DOMEvent, itemID int64, props *HappeningListProps) {
	// Handle delete confirmation navigation
	if props.DeletingItemID == itemID {
		keyEvent := e.KeydownEvent
		switch keyEvent.KeyType {
		case dom.KeyTypeLeft:
			if props.OnNavigateDeleteConfirm != nil {
				props.OnNavigateDeleteConfirm(-1)
			}
		case dom.KeyTypeRight:
			if props.OnNavigateDeleteConfirm != nil {
				props.OnNavigateDeleteConfirm(1)
			}
		case dom.KeyTypeEnter:
			if props.DeleteConfirmButton == 0 {
				// Delete button selected
				if props.OnConfirmDelete != nil {
					props.OnConfirmDelete(e, itemID)
				}
			} else {
				// Cancel button selected
				if props.OnCancelDelete != nil {
					props.OnCancelDelete(e)
				}
			}
		case dom.KeyTypeEsc:
			if props.OnCancelDelete != nil {
				props.OnCancelDelete(e)
			}
		}
		return
	}

	// Handle normal key events
	keyEvent := e.KeydownEvent
	switch keyEvent.KeyType {
	default:
		key := string(keyEvent.Runes)
		switch key {
		case "d":
			if props.OnDeleteItem != nil {
				props.OnDeleteItem(itemID)
			}
		case "e":
			if props.OnEditItem != nil {
				props.OnEditItem(itemID)
			}
		case "/":
			// Focus input when "/" is pressed and clear item focus
			if props.OnBlurItem != nil {
				props.OnBlurItem(itemID)
			}
			if props.InputState != nil {
				props.InputState.Focused = true
			}
		}
	}
}

// GetSampleHappenings returns sample happening data for testing
func GetSampleHappenings() []*models.Happening {
	// Simulate network latency
	time.Sleep(200 * time.Millisecond)

	now := time.Now()

	return []*models.Happening{
		{
			ID:         1,
			Content:    "Started working on the new feature",
			CreateTime: now.Add(-2 * time.Hour),
		},
		{
			ID:         2,
			Content:    "Had a great meeting with the team",
			CreateTime: now.Add(-1 * 24 * time.Hour), // 1 day ago
		},
		{
			ID:         3,
			Content:    "Completed the project milestone",
			CreateTime: now.Add(-3 * 24 * time.Hour), // 3 days ago
		},
		{
			ID:         4,
			Content:    "Learned something new about Go",
			CreateTime: now.Add(-7 * 24 * time.Hour), // 1 week ago
		},
		{
			ID:         5,
			Content:    "Started this todo application",
			CreateTime: now.Add(-365 * 24 * time.Hour), // 1 year ago
		},
	}
}
