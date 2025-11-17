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
		cloneEntry := *entry
		ok := fn(&cloneEntry)
		// Recursively filter children
		filteredChildren := FilterEntries(cloneEntry.Children, fn)

		// Include entry if it matches or has matching children
		if ok || len(filteredChildren) > 0 {
			// Create a copy of the entry with filtered children
			cloneEntry.Children = filteredChildren
			filtered = append(filtered, &cloneEntry)
		}
	}

	return filtered
}

// FilterEntriesQuery filters entries and their children based on search query
// Also searches within notes and highlights matched parts
func FilterEntriesQuery(entries []*models.LogEntryView, query string) []*models.LogEntryView {
	query = strings.ToLower(query)

	return FilterEntries(entries, func(entry *models.LogEntryView) bool {
		// Search in entry text
		if query == "" {
			entry.MatchTexts = nil
			return true
		}

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

		cloneNotes := make([]*models.NoteView, len(entry.Notes))
		for i, note := range entry.Notes {
			cloneNote := *note

			noteTextIdx := strings.Index(strings.ToLower(cloneNote.Data.Text), query)
			var noteMatchTexts []models.MatchText
			if noteTextIdx >= 0 {
				noteMatchTexts = []models.MatchText{
					{
						Text: cloneNote.Data.Text[:noteTextIdx],
					},
					{
						Text:  cloneNote.Data.Text[noteTextIdx : noteTextIdx+len(query)],
						Match: true,
					},
					{
						Text: cloneNote.Data.Text[noteTextIdx+len(query):],
					},
				}
				hasNoteMatch = true
			}
			cloneNote.MatchTexts = noteMatchTexts
			cloneNotes[i] = &cloneNote
		}

		entry.Notes = cloneNotes

		// Return true if either entry text or any note matches
		return entryTextIdx >= 0 || hasNoteMatch
	})
}
