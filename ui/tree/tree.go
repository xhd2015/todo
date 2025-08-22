package tree

import (
	"bytes"

	"github.com/xhd2015/todo/models"
)

func RenderEntriesString(entries []*models.LogEntryView) string {
	var b bytes.Buffer
	// renderEntriesOut(&b, entries)
	// return b.String()

	renderEntries(entries, "", false, func(prefix string, connector string, entry *models.LogEntryView) {
		// Determine the symbol based on completion status
		symbol := "•"
		if entry.Data.Done {
			symbol = "✓"
		}
		b.WriteString(prefix + connector + symbol + " " + entry.Data.Text + "\n")
	})
	return b.String()
}

func RenderEntries(entries []*models.LogEntryView, callback func(prefix string, connector string, entry *models.LogEntryView)) {
	renderEntriesWithState(entries, "", false, callback)
}

func renderEntries(entries []*models.LogEntryView, prefix string, hasVerticalLine bool, callback func(prefix string, connector string, entry *models.LogEntryView)) {
	renderEntriesWithState(entries, prefix, hasVerticalLine, callback)
}

func renderEntriesWithState(entries []*models.LogEntryView, prefix string, hasVerticalLine bool, callback func(prefix string, connector string, entry *models.LogEntryView)) {
	for i, entry := range entries {
		isLast := i == len(entries)-1

		// Write the current entry
		var connector string
		if prefix != "" {
			connector = getConnector(isLast)
		}
		callback(prefix, connector, entry)

		// Render children with appropriate prefix
		if len(entry.Children) > 0 {
			childPrefix, childHasVerticalLine := CalculateChildPrefix(prefix, isLast, hasVerticalLine)
			renderEntriesWithState(entry.Children, childPrefix, childHasVerticalLine, callback)
		}
	}
}
