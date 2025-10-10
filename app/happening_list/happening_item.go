package happening_list

import (
	"fmt"
	"time"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models"
)

type HappeningItemProps struct {
	Item    *models.Happening
	Focused bool
	OnFocus func()
	OnBlur  func()
}

// HappeningItem renders a single happening item with main text and relative time
func HappeningItem(data *HappeningItemProps) *dom.Node {
	relativeTime := formatRelativeTime(data.Item.CreateTime)

	// Style for main text - add color when focused
	mainTextStyle := styles.Style{
		// Color: "white",
	}
	if data.Focused {
		mainTextStyle.Color = "#00FF00" // Green color when focused
		mainTextStyle.Bold = true
	}

	return dom.Div(
		dom.DivProps{
			Focusable: true,
			Focused:   data.Focused,
			OnFocus:   data.OnFocus,
			OnBlur:    data.OnBlur,
		},
		// Main text line with bullet indicator
		dom.Text("• "+data.Item.Content, mainTextStyle),
		// Subtitle line with relative time (indented to align with main text)
		dom.Br(),
		dom.Text("  "+relativeTime, styles.Style{
			Color:  "grey", // Gray color for subtitle
			Italic: true,
		}),
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
