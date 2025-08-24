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
// Also searches within notes and highlights matched parts
func FilterEntriesQuery(entries []*models.LogEntryView, query string) []*models.LogEntryView {
	if query == "" {
		return entries
	}

	query = strings.ToLower(query)

	return FilterEntries(entries, func(entry *models.LogEntryView) bool {
		// Search in entry text
		entryTextIdx := strings.Index(strings.ToLower(entry.Data.Text), query)
		var matchTexts []models.MatchText
		if entryTextIdx >= 0 {
			matchTexts = []models.MatchText{
				{
					Text: entry.Data.Text[:entryTextIdx],
				},
				{
					Text:  entry.Data.Text[entryTextIdx : entryTextIdx+len(query)],
					Match: true,
				},
				{
					Text: entry.Data.Text[entryTextIdx+len(query):],
				},
			}
		}
		entry.MatchTexts = matchTexts

		// Search in notes
		hasNoteMatch := false
		for _, note := range entry.Notes {
			noteTextIdx := strings.Index(strings.ToLower(note.Data.Text), query)
			var noteMatchTexts []models.MatchText
			if noteTextIdx >= 0 {
				noteMatchTexts = []models.MatchText{
					{
						Text: note.Data.Text[:noteTextIdx],
					},
					{
						Text:  note.Data.Text[noteTextIdx : noteTextIdx+len(query)],
						Match: true,
					},
					{
						Text: note.Data.Text[noteTextIdx+len(query):],
					},
				}
				hasNoteMatch = true
			}
			note.MatchTexts = noteMatchTexts
		}

		// Return true if either entry text or any note matches
		return entryTextIdx >= 0 || hasNoteMatch
	})
}
