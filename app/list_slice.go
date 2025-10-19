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

type EntryOptions struct {
	MaxEntries      int
	SliceStart      int
	SelectedID      int64
	SelectedSource  SelectedSource
	ZenMode         bool
	SearchActive    bool
	Query           string
	ShowNotes       bool
	FocusingEntryID int64
	ExpandAll       bool
}

func computeVisibleEntries(entries models.LogEntryViews, opts EntryOptions) ComputeResult {
	var focusedRootPath []string
	// Process focusing if focusingEntryID is provided
	if opts.FocusingEntryID != 0 {
		entries, focusedRootPath = processFocusedEntries(entries, opts.FocusingEntryID)
	}

	// Filter entries based on search query if active
	entriesToRender := applyFilter(entries, opts.ZenMode, opts.SearchActive, opts.Query)

	// Process collapsed entries to hide children and add count information
	processCollapsedEntries(entriesToRender, opts.ExpandAll)

	// Add top-level entries (ParentID == 0)
	topLevelEntries := make([]*models.LogEntryView, 0)
	for _, entry := range entriesToRender {
		if entry.Data.ParentID == 0 {
			topLevelEntries = append(topLevelEntries, entry)
		}
	}

	var flatEntries []TreeEntry

	// Add focused root path as the first entry if in focused mode
	if len(focusedRootPath) > 0 {
		focusedTreeEntry := TreeEntry{
			Type:   TreeEntryType_FocusedItem,
			Prefix: "",
			IsLast: false,
			FocusedItem: &TreeFocusedItem{
				RootPath: focusedRootPath,
			},
		}
		flatEntries = append(flatEntries, focusedTreeEntry)
	}

	for i, entry := range topLevelEntries {
		isLast := i == len(topLevelEntries)-1
		flatEntries = addEntryRecursive(flatEntries, entry, 0, "", isLast, false, opts.ShowNotes)
	}
	entriesAbove, entriesBelow, visibleEntries, effectiveSliceStart := sliceEntries(flatEntries, opts.MaxEntries, opts.SliceStart, opts.SelectedID, opts.SelectedSource)
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

// processFocusedEntries processes entries for focused mode, showing only the focused entry and its subtree
func processFocusedEntries(entries models.LogEntryViews, focusingEntryID int64) (models.LogEntryViews, []string) {
	// Find the focused entry and build its root path
	focusedEntry := findEntryInTree(entries, focusingEntryID)
	if focusedEntry == nil {
		// If focused entry not found, return original entries
		return entries, nil
	}

	// Build the root path for the focused entry
	rootPath := buildRootPath(entries, focusingEntryID)

	// should clear parent entry id
	children := focusedEntry.Children

	copiedChildren := make(models.LogEntryViews, len(children))
	for i, child := range children {
		p := *child
		p.Data.ParentID = 0
		copiedChildren[i] = &p
	}

	return focusedEntry.Children, rootPath
}

// findEntryInTree recursively finds an entry by ID in the tree
func findEntryInTree(entries models.LogEntryViews, entryID int64) *models.LogEntryView {
	for _, entry := range entries {
		if entry.Data.ID == entryID {
			return entry
		}
		if found := findEntryInTree(entry.Children, entryID); found != nil {
			return found
		}
	}
	return nil
}

// buildRootPath builds the path from root to the specified entry
func buildRootPath(entries models.LogEntryViews, entryID int64) []string {
	path := findPathToEntry(entries, entryID, []string{})
	return path
}

// findPathToEntry recursively finds the path to an entry
func findPathToEntry(entries models.LogEntryViews, entryID int64, currentPath []string) []string {
	for _, entry := range entries {
		newPath := append(currentPath, entry.Data.Text)
		if entry.Data.ID == entryID {
			return newPath
		}
		if found := findPathToEntry(entry.Children, entryID, newPath); found != nil {
			return found
		}
	}
	return nil
}

// processCollapsedEntries processes the tree to hide children of collapsed entries
// and adds collapsed count information to the entry view
// If expandAll is true, ignores collapse flags and shows all entries
func processCollapsedEntries(entries models.LogEntryViews, expandAll bool) {
	for _, entry := range entries {
		if !expandAll && entry.Data.Collapsed {
			// If we have children visible, we need to collapse them
			if len(entry.Children) > 0 {
				// Count total children (including nested children)
				collapsedCount := countAllChildren(entry.Children)

				// Store the original children for potential expansion later
				// and clear the visible children
				entry.CollapsedChildren = entry.Children
				entry.CollapsedCount = collapsedCount
				entry.Children = []*models.LogEntryView{}
			}
			// If children are already hidden but we don't have a count, keep existing state
		} else {
			// If not collapsed but we have collapsed children, restore them
			if len(entry.CollapsedChildren) > 0 {
				entry.Children = entry.CollapsedChildren
				entry.CollapsedChildren = nil
				entry.CollapsedCount = 0
			}
			// Recursively process non-collapsed children
			processCollapsedEntries(entry.Children, expandAll)
		}
	}
}

// countAllChildren recursively counts all children and their descendants
func countAllChildren(children []*models.LogEntryView) int {
	count := len(children)
	for _, child := range children {
		count += countAllChildren(child.Children)
	}
	return count
}
