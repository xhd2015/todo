package search

import (
	"strings"

	"github.com/xhd2015/todo/models"
)

func FilterEntries(entries []*models.LogEntryView, fn func(entry *models.LogEntryView) bool) []*models.LogEntryView {
	if fn == nil {
		return entries
	}
	var filtered []*models.LogEntryView
	for _, entry := range entries {
		ok := fn(entry)
		// Recursively filter children
		filteredChildren := FilterEntries(entry.Children, fn)

		// Include entry if it matches or has matching children
		if ok || len(filteredChildren) > 0 {
			// Create a copy of the entry with filtered children
			filteredEntry := *entry
			filteredEntry.Children = filteredChildren
			filtered = append(filtered, &filteredEntry)
		}
	}

	return filtered
}

// FilterEntriesQuery filters entries and their children based on search query
func FilterEntriesQuery(entries []*models.LogEntryView, query string) []*models.LogEntryView {
	if query == "" {
		return entries
	}

	query = strings.ToLower(query)

	return FilterEntries(entries, func(entry *models.LogEntryView) bool {
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
		return idx >= 0
	})
}
