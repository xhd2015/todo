package app

import (
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/ui/search"
	"github.com/xhd2015/todo/ui/tree"
)

type ComputeResult struct {
	EntriesAbove        int
	EntriesBelow        int
	VisibleEntries      []TreeEntry
	FullEntries         []TreeEntry
	EffectiveSliceStart int
}

func computeVisibleEntries(entries models.LogEntryViews, maxEntries int, sliceStart int, selectedID int64, selectedSource SelectedSource, zenMode bool, searchActive bool, query string, showNotes bool) ComputeResult {
	// Filter entries based on search query if active
	entriesToRender := applyFilter(entries, zenMode, searchActive, query)

	// Add top-level entries (ParentID == 0)
	topLevelEntries := make([]*models.LogEntryView, 0)
	for _, entry := range entriesToRender {
		if entry.Data.ParentID == 0 {
			topLevelEntries = append(topLevelEntries, entry)
		}
	}

	var flatEntries []TreeEntry
	for i, entry := range topLevelEntries {
		isLast := i == len(topLevelEntries)-1
		flatEntries = addEntryRecursive(flatEntries, entry, 0, "", isLast, false, showNotes)
	}
	entriesAbove, entriesBelow, visibleEntries, effectiveSliceStart := sliceEntries(flatEntries, maxEntries, sliceStart, selectedID, selectedSource)
	return ComputeResult{
		EntriesAbove:        entriesAbove,
		EntriesBelow:        entriesBelow,
		VisibleEntries:      visibleEntries,
		FullEntries:         flatEntries,
		EffectiveSliceStart: effectiveSliceStart,
	}
}

func addEntryRecursive(flatEntries []TreeEntry, entry *models.LogEntryView, depth int, prefix string, isLast bool, hasVerticalLine bool, globalShowNotes bool) []TreeEntry {
	// Implement 'v implies n': if IncludeHistory is true, also show notes
	// Also show notes if global notes mode is enabled
	showNotes := entry.IncludeNotes || entry.IncludeHistory || globalShowNotes
	return addEntryRecursiveWithHistory(flatEntries, entry, depth, prefix, isLast, hasVerticalLine, entry.IncludeHistory, showNotes, globalShowNotes)
}

func addEntryRecursiveWithHistory(flatEntries []TreeEntry, entry *models.LogEntryView, depth int, prefix string, isLast bool, hasVerticalLine bool, showNotesInSubtree bool, showNotesFromParent bool, globalShowNotes bool) []TreeEntry {
	// Add this entry
	flatEntries = append(flatEntries, TreeEntry{
		Type:   TreeEntryType_Log,
		Prefix: prefix,
		IsLast: isLast,
		Log: &TreeLog{
			Entry: entry,
		},
	})

	// Add notes based on different conditions
	if len(entry.Notes) > 0 {
		// Check if we should show notes due to explicit flags
		showAllNotes := showNotesFromParent || entry.IncludeNotes

		// If not showing all notes, check if any notes have search matches
		var notesToShow []*models.NoteView
		if showAllNotes {
			// Show all notes
			notesToShow = entry.Notes
		} else {
			// Only show notes that have search matches
			for _, note := range entry.Notes {
				if len(note.MatchTexts) > 0 {
					notesToShow = append(notesToShow, note)
				}
			}
		}

		// Render the notes that should be shown
		for i, note := range notesToShow {
			notePrefix := prefix
			if !isLast {
				notePrefix += "│ "
			} else {
				notePrefix += "  "
			}

			isLastNote := i == len(notesToShow)-1 && len(entry.Children) == 0
			flatEntries = append(flatEntries, TreeEntry{
				Type:   TreeEntryType_Note,
				Prefix: notePrefix,
				IsLast: isLastNote,
				Note: &TreeNote{
					Note:    note,
					EntryID: entry.Data.ID,
				},
			})
		}
	}

	// Add children recursively, passing down the flags
	for childIndex, child := range entry.Children {
		isLastChild := (childIndex == len(entry.Children)-1)
		// Calculate child prefix using the common utility function
		childPrefix, childHasVerticalLine := tree.CalculateChildPrefix(prefix, isLast, hasVerticalLine)
		// Pass down showNotesInSubtree if current entry has IncludeHistory, or inherit from parent
		childShowHistory := showNotesInSubtree || child.IncludeHistory
		// Pass down showNotesFromParent if current entry has IncludeNotes or IncludeHistory, or inherit from parent
		// Implement 'v implies n': if parent has IncludeHistory, also show notes for children
		// Also show notes if global notes mode is enabled
		childShowNotes := showNotesFromParent || entry.IncludeNotes || entry.IncludeHistory || globalShowNotes
		flatEntries = addEntryRecursiveWithHistory(flatEntries, child, depth+1, childPrefix, isLastChild, childHasVerticalLine, childShowHistory, childShowNotes, globalShowNotes)
	}

	return flatEntries
}

func applyFilter(list models.LogEntryViews, zenMode bool, searchActive bool, query string) models.LogEntryViews {
	// Filter entries based on search query if active
	entriesToRender := list

	if zenMode {
		entriesToRender = search.FilterEntries(list, func(entry *models.LogEntryView) bool {
			return entry.Data.HighlightLevel > 0 && !entry.Data.Done
		})
	}

	if searchActive && query != "" {
		entriesToRender = search.FilterEntriesQuery(entriesToRender, query)
	}

	return entriesToRender
}

func sliceEntries(entries []TreeEntry, maxEntries int, sliceStart int, selectedID int64, selectedFromSource SelectedSource) (int, int, []TreeEntry, int) {
	if maxEntries <= 0 || len(entries) <= maxEntries {
		if sliceStart == -1 {
			sliceStart = 0
		}
		return 0, 0, entries, sliceStart
	}
	totalEntries := len(entries)
	var visibleEntries []TreeEntry
	// Ensure SliceStart is within bounds

	// default to last N entries
	if sliceStart == -1 {
		sliceStart = totalEntries - maxEntries
	}
	if sliceStart < 0 {
		sliceStart = 0
	}
	if sliceStart >= totalEntries {
		sliceStart = totalEntries - maxEntries
		if sliceStart < 0 {
			sliceStart = 0
		}
	}

	end := sliceStart + maxEntries
	if end > totalEntries {
		end = totalEntries
	}

	if selectedID != 0 {
		var foundIndex int = -1
		for i, wrapperEntry := range entries {
			if wrapperEntry.Type == TreeEntryType_Log && wrapperEntry.Log != nil && wrapperEntry.Log.Entry.Data.ID == selectedID {
				foundIndex = i
				break
			}
		}
		if foundIndex != -1 {
			switch selectedFromSource {
			case SelectedSource_Search:
				// on top
				sliceStart = foundIndex
				end = foundIndex + maxEntries
				if end > totalEntries {
					end = totalEntries
				}
			case SelectedSource_NavigateByKey:
				// ensure the selected entry is visible
				if foundIndex < sliceStart {
					sliceStart = foundIndex
					end = sliceStart + maxEntries
				} else if foundIndex >= end {
					sliceStart = foundIndex - maxEntries + 1
					end = sliceStart + maxEntries
				}
			default:
				if foundIndex < sliceStart {
					sliceStart = foundIndex
					end = sliceStart + maxEntries
				} else if foundIndex >= end {
					sliceStart = foundIndex - maxEntries + 1
					end = sliceStart + maxEntries
				}
			}
		}
	}

	visibleEntries = entries[sliceStart:end]

	// Calculate exact counts for indicators
	entriesAbove := sliceStart
	entriesBelow := totalEntries - end
	return entriesAbove, entriesBelow, visibleEntries, sliceStart
}
