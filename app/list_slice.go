package app

import (
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/ui/search"
)

type ComputeResult struct {
	EntriesAbove   int
	EntriesBelow   int
	VisibleEntries []EntryWithDepth
	FullEntries    []EntryWithDepth
}

func computeVisibleEntries(entries models.LogEntryViews, maxEntries int, sliceStart int, zenMode bool, searchActive bool, query string) ComputeResult {
	// Filter entries based on search query if active
	entriesToRender := applyFilter(entries, zenMode, searchActive, query)

	// Add top-level entries (ParentID == 0)
	topLevelEntries := make([]*models.LogEntryView, 0)
	for _, entry := range entriesToRender {
		if entry.Data.ParentID == 0 {
			topLevelEntries = append(topLevelEntries, entry)
		}
	}

	var flatEntries []EntryWithDepth
	for _, entry := range topLevelEntries {
		flatEntries = addEntryRecursive(flatEntries, entry, 0, []bool{})
	}
	entriesAbove, entriesBelow, visibleEntries := sliceEntries(flatEntries, maxEntries, sliceStart)
	return ComputeResult{
		EntriesAbove:   entriesAbove,
		EntriesBelow:   entriesBelow,
		VisibleEntries: visibleEntries,
		FullEntries:    flatEntries,
	}
}

func addEntryRecursive(flatEntries []EntryWithDepth, entry *models.LogEntryView, depth int, ancestorIsLast []bool) []EntryWithDepth {
	flatEntries = append(flatEntries, EntryWithDepth{
		Entry:       entry,
		Depth:       depth,
		IsLastChild: ancestorIsLast,
	})

	// Add children recursively
	for childIndex, child := range entry.Children {
		isLastChild := (childIndex == len(entry.Children)-1)
		// Create ancestor info for child: copy parent's info and add current level
		childAncestorIsLast := make([]bool, depth+1)
		copy(childAncestorIsLast, ancestorIsLast)
		childAncestorIsLast[depth] = isLastChild
		flatEntries = addEntryRecursive(flatEntries, child, depth+1, childAncestorIsLast)
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

func sliceEntries(entries []EntryWithDepth, maxEntries int, sliceStart int) (int, int, []EntryWithDepth) {
	if maxEntries <= 0 || len(entries) <= maxEntries {
		return 0, 0, entries
	}
	totalEntries := len(entries)
	var visibleEntries []EntryWithDepth
	// Ensure SliceStart is within bounds
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

	visibleEntries = entries[sliceStart:end]

	// Calculate exact counts for indicators
	entriesAbove := sliceStart
	entriesBelow := totalEntries - end
	return entriesAbove, entriesBelow, visibleEntries
}
