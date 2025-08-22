package tree

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/xhd2015/todo/models"
)

func RenderItem(entry *models.LogEntryView, showID bool, renderStrikeThrough bool) string {
	var strikethroughStyle = lipgloss.NewStyle().Strikethrough(true)
	// Choose bullet based on completion status
	bullet := "•"
	if entry.Data.Done {
		bullet = "✓"
	}

	// Apply styling
	text := entry.Data.Text
	if entry.Data.Done && renderStrikeThrough {
		text = strikethroughStyle.Render(text)
	}

	// Add visibility indicator if children are visible
	visibilityIndicator := ""
	if entry.ChildrenVisible {
		visibilityIndicator = " (*)"
	}

	var idIndicator string
	if showID {
		idIndicator = fmt.Sprintf(" (%d)", entry.Data.ID)
	}

	return bullet + " " + text + visibilityIndicator + idIndicator
}
