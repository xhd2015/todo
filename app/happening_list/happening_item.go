package happening_list

import (
	"fmt"
	"time"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/component"
	"github.com/xhd2015/todo/models"
)

type HappeningItemProps struct {
	Item    *models.Happening
	Focused bool
	OnFocus func()
	OnBlur  func()

	// Edit/Delete functionality
	IsEditing               bool
	IsDeleting              bool
	EditInputState          *models.InputState
	DeleteConfirmButton     int
	OnEdit                  func()
	OnDelete                func()
	OnSaveEdit              func(content string)
	OnCancelEdit            func(e *dom.DOMEvent)
	OnConfirmDelete         func(e *dom.DOMEvent)
	OnCancelDelete          func(e *dom.DOMEvent)
	OnNavigateDeleteConfirm func(direction int) // -1 for left, 1 for right
}

// HappeningItem renders a single happening item with main text and relative time
func HappeningItem(data *HappeningItemProps) *dom.Node {
	relativeTime := formatRelativeTime(data.Item.CreateTime)

	// If in editing mode, show input field
	if data.IsEditing && data.EditInputState != nil {
		return dom.Div(
			dom.DivProps{},
			component.SearchInput(component.InputProps{
				Placeholder: "edit happening",
				State:       data.EditInputState,
				Width:       50,
				OnKeyDown: func(event *dom.DOMEvent) bool {
					keyEvent := event.KeydownEvent
					switch keyEvent.KeyType {
					case dom.KeyTypeEnter:
						if data.OnSaveEdit != nil {
							data.OnSaveEdit(data.EditInputState.Value)
						}
						return true // Event handled, prevent further processing
					case dom.KeyTypeEsc:
						if data.OnCancelEdit != nil {
							data.OnCancelEdit(event)
						}
						return true // Event handled, prevent further processing
					case dom.KeyTypeCtrlC:
						if data.OnCancelEdit != nil {
							data.OnCancelEdit(event)
						}
						return true // Event handled, prevent further processing
					}
					return false // Event not handled, allow normal input processing (including backspace)
				},
				OnEnter: func(s string) bool {
					// This should not be called since we handle Enter in OnKeyDown
					return false
				},
			}),
			dom.Br(),
			dom.Text("  "+relativeTime, styles.Style{
				Color:  "grey",
				Italic: true,
			}),
		)
	}

	// Style for main text - add color when focused
	mainTextStyle := styles.Style{
		// Color: "white",
	}
	if data.Focused {
		mainTextStyle.Color = "#00FF00" // Green color when focused
		mainTextStyle.Bold = true
	}

	var children []*dom.Node

	// Main content
	children = append(children,
		// Main text line with bullet indicator
		dom.Text("• "+data.Item.Content, mainTextStyle),
		// Subtitle line with relative time (indented to align with main text)
		dom.Br(),
		dom.Text("  "+relativeTime, styles.Style{
			Color:  "grey", // Gray color for subtitle
			Italic: true,
		}),
	)

	// If in deleting mode, show confirmation dialog
	if data.IsDeleting {
		children = append(children,
			dom.Br(),
			renderDeleteConfirm(data),
		)
	}

	return dom.Div(
		dom.DivProps{
			Focusable: true,
			Focused:   data.Focused,
			OnFocus:   data.OnFocus,
			OnBlur:    data.OnBlur,
			OnKeyDown: func(e *dom.DOMEvent) {
				// Handle delete confirmation navigation
				if data.IsDeleting {
					keyEvent := e.KeydownEvent
					switch keyEvent.KeyType {
					case dom.KeyTypeLeft:
						if data.OnNavigateDeleteConfirm != nil {
							data.OnNavigateDeleteConfirm(-1)
						}
					case dom.KeyTypeRight:
						if data.OnNavigateDeleteConfirm != nil {
							data.OnNavigateDeleteConfirm(1)
						}
					case dom.KeyTypeEnter:
						if data.DeleteConfirmButton == 0 {
							// Delete button selected
							if data.OnConfirmDelete != nil {
								data.OnConfirmDelete(e)
							}
						} else {
							// Cancel button selected
							if data.OnCancelDelete != nil {
								data.OnCancelDelete(e)
							}
						}
					case dom.KeyTypeEsc:
						if data.OnCancelDelete != nil {
							data.OnCancelDelete(e)
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
						if data.OnDelete != nil {
							data.OnDelete()
						}
					case "e":
						if data.OnEdit != nil {
							data.OnEdit()
						}
					}
				}
			},
		},
		children...,
	)
}

// renderDeleteConfirm renders the delete confirmation dialog
func renderDeleteConfirm(data *HappeningItemProps) *dom.Node {
	deleteStyle := styles.Style{Color: "red"}
	cancelStyle := styles.Style{Color: "white"}

	// Highlight selected button
	if data.DeleteConfirmButton == 0 {
		deleteStyle.Bold = true
		deleteStyle.Color = "#FF0000" // Bright red when selected
	} else {
		cancelStyle.Bold = true
		cancelStyle.Color = "#00FF00" // Green when selected
	}

	return dom.Div(
		dom.DivProps{},
		dom.Text("  Delete happening? ", styles.Style{Color: "yellow"}),
		dom.Text("[Delete]", deleteStyle),
		dom.Text(" ", styles.Style{}),
		dom.Text("[Cancel]", cancelStyle),
	)
}

// formatRelativeTime formats a timestamp as relative time (e.g., "1h ago", "2d ago", "1 year ago")
func formatRelativeTime(timestamp time.Time) string {
	now := time.Now()
	duration := now.Sub(timestamp)

	if duration < 0 {
		return "in the future"
	}

	seconds := int(duration.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24
	weeks := days / 7
	months := days / 30
	years := days / 365

	switch {
	case seconds < 60:
		if seconds <= 1 {
			return "just now"
		}
		return fmt.Sprintf("%ds ago", seconds)
	case minutes < 60:
		if minutes == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", minutes)
	case hours < 24:
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	case days < 7:
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	case weeks < 4:
		if weeks == 1 {
			return "1w ago"
		}
		return fmt.Sprintf("%dw ago", weeks)
	case months < 12:
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}
