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
		idx := strings.Index(strings.ToLower(entry.Data.Text), query)
		var matchTexts []models.MatchText
		if idx >= 0 {
			matchTexts = []models.MatchText{
				{
					Text: entry.Data.Text[:idx],
				},
				{
					Text:  entry.Data.Text[idx : idx+len(query)],
					Match: true,
				},
				{
					Text: entry.Data.Text[idx+len(query):],
				},
			}
		}
		entry.MatchTexts = matchTexts

		// Recursively filter children
		filteredChildren := FilterEntriesRecursive(entry.Children, query)

		// Include entry if it matches or has matching children
		if idx >= 0 || len(filteredChildren) > 0 {
			// Create a copy of the entry with filtered children
			filteredEntry := *entry
			filteredEntry.Children = filteredChildren
			filtered = append(filtered, &filteredEntry)
		}
	}

	return filtered
}
