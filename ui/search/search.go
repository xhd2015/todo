package search

import (
	"strings"

	"github.com/xhd2015/todo/models"
)

// FilterEntriesRecursive filters entries and their children based on search query
func FilterEntriesRecursive(entries []*models.LogEntryView, query string) []*models.LogEntryView {
	if query == "" {
		return entries
	}

	query = strings.ToLower(query)
	var filtered []*models.LogEntryView

	for _, entry := range entries {
		// Check if current entry matches
		entryMatches := strings.Contains(strings.ToLower(entry.Data.Text), query)

		// Recursively filter children
		filteredChildren := FilterEntriesRecursive(entry.Children, query)

		// Include entry if it matches or has matching children
		if entryMatches || len(filteredChildren) > 0 {
			// Create a copy of the entry with filtered children
			filteredEntry := &models.LogEntryView{
				Data:       entry.Data,
				DetailPage: entry.DetailPage,
				Notes:      entry.Notes,
				Children:   filteredChildren,
			}
			filtered = append(filtered, filteredEntry)
		}
	}

	return filtered
}
