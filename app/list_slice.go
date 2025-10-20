package app

import (
	"github.com/xhd2015/todo/app/exp"
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/ui/search"
	"github.com/xhd2015/todo/ui/tree"
)

// Group IDs for organizing entries in group mode
const (
	GROUP_DEADLINE_ID    = 1
	GROUP_WORKPERF_ID    = 2
	GROUP_LIFEENHANCE_ID = 3
	GROUP_WORKHACK_ID    = 4
	GROUP_LIFEHACK_ID    = 5
	GROUP_OTHER_ID       = 6
)

// LogAndGroup tracks the group association and staging children for an entry during group organization
type LogAndGroup struct {
	LogGroupID      int64
	StagingChildren models.LogEntryViews
}

// mountToGroup mounts a log entry to its appropriate group based on mapping and parent chain
// Returns a cloned entry with proper group association and children
func mountToGroup(
	entry *models.LogEntryView,
	parents []*LogAndGroup,
	mapping map[int64]int64,
	normalGroups models.LogEntryViews,
	otherGroup *models.LogEntryView,
) *models.LogEntryView {
	groupID := mapping[entry.Data.ID]
	var foundNormalGroup *models.LogEntryView
	for _, groupEntry := range normalGroups {
		if groupEntry.Data.ID == groupID {
			foundNormalGroup = groupEntry
			break
		}
	}

	targetGroup := otherGroup
	if foundNormalGroup != nil {
		targetGroup = foundNormalGroup
	}
	targetGroupID := targetGroup.Data.ID

	var foundSameGroupParent *LogAndGroup
	n := len(parents)
	for i := n - 1; i >= 0; i-- {
		p := parents[i]
		if p.LogGroupID == targetGroupID {
			foundSameGroupParent = p
			break
		}
	}
	var logAndGroup *LogAndGroup

	cloneEntry := *entry
	if foundSameGroupParent == nil {
		targetGroup.Children = append(targetGroup.Children, &cloneEntry)
		logAndGroup = &LogAndGroup{
			LogGroupID: targetGroupID,
		}
	} else {
		foundSameGroupParent.StagingChildren = append(foundSameGroupParent.StagingChildren, &cloneEntry)
		logAndGroup = &LogAndGroup{
			LogGroupID: foundSameGroupParent.LogGroupID,
		}
	}
	parents = append(parents, logAndGroup)

	for _, child := range entry.Children {
		mountToGroup(child, parents, mapping, normalGroups, otherGroup)
	}

	cloneEntry.Children = logAndGroup.StagingChildren
	return &cloneEntry
}

type ComputeResult struct {
	EntriesAbove        int
	EntriesBelow        int
	VisibleEntries      []TreeEntry
	FullEntries         []TreeEntry
	EffectiveSliceStart int
}

type EntryOptions struct {
	MaxEntries         int
	SliceStart         int
	SelectedID         int64
	SelectedSource     SelectedSource
	ZenMode            bool
	SearchActive       bool
	Query              string
	ShowNotes          bool
	FocusingEntryID    models.EntryIdentity
	ExpandAll          bool
	ViewMode           ViewMode
	GroupCollapseState map[int64]bool
}

func computeVisibleEntries(entries models.LogEntryViews, opts EntryOptions) ComputeResult {
	var focusedRootPath []string
	// Process focusing if focusingEntryID is provided
	if opts.FocusingEntryID.IsSet() {
		entries, focusedRootPath = processFocusedEntries(entries, opts.FocusingEntryID)
	}

	// Organize entries into groups if ViewMode is Group
	if opts.ViewMode == ViewMode_Group {
		entries = organizeEntriesIntoGroups(entries, opts.GroupCollapseState)
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
			Type:   models.LogEntryViewType_FocusedItem,
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
	// Determine entry type based on the ViewType field
	var treeEntry TreeEntry

	if entry.ViewType == models.LogEntryViewType_Group {
		treeEntry = TreeEntry{
			Type:   models.LogEntryViewType_Group,
			Prefix: prefix,
			IsLast: isLast,
			Entry:  entry,
			Group: &TreeGroup{
				ID:   entry.Data.ID,
				Name: entry.Data.Text,
			},
		}
	} else {
		treeEntry = TreeEntry{
			Type:   models.LogEntryViewType_Log,
			Prefix: prefix,
			IsLast: isLast,
			Entry:  entry,
			Log:    &TreeLog{},
		}
	}

	// Add this entry
	flatEntries = append(flatEntries, treeEntry)

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
				Type:   models.LogEntryViewType_Note,
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
			if wrapperEntry.Type == models.LogEntryViewType_Log && wrapperEntry.Log != nil && wrapperEntry.Entry.Data.ID == selectedID {
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
func processFocusedEntries(entries models.LogEntryViews, focusingEntryID models.EntryIdentity) (models.LogEntryViews, []string) {
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
func findEntryInTree(entries models.LogEntryViews, targetEntry models.EntryIdentity) *models.LogEntryView {
	for _, entry := range entries {
		if entry.ViewType == targetEntry.EntryType && entry.Data.ID == targetEntry.ID {
			return entry
		}
		if found := findEntryInTree(entry.Children, targetEntry); found != nil {
			return found
		}
	}
	return nil
}

// buildRootPath builds the path from root to the specified entry
func buildRootPath(entries models.LogEntryViews, entryID models.EntryIdentity) []string {
	path := findPathToEntry(entries, entryID, []string{})
	return path
}

// findPathToEntry recursively finds the path to an entry
func findPathToEntry(entries models.LogEntryViews, targetEntry models.EntryIdentity, currentPath []string) []string {
	for _, entry := range entries {
		newPath := append(currentPath, entry.Data.Text)
		if entry.SameIdentity(targetEntry) {
			return newPath
		}
		if found := findPathToEntry(entry.Children, targetEntry, newPath); found != nil {
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

// organizeEntriesIntoGroups organizes entries into pseudo groups based on their properties
func organizeEntriesIntoGroups(entries models.LogEntryViews, groupCollapseState map[int64]bool) models.LogEntryViews {
	// Create group entries in order: Deadline, WorkPerf, LifeEnhance, WorkHack, LifeHack, Other
	groupNames := []string{"Deadline", "WorkPerf", "LifeEnhance", "WorkHack", "LifeHack", "Other"}
	groupEntries := make(models.LogEntryViews, 0, len(groupNames))

	for i, name := range groupNames {
		id := int64(i + 1) // Natural ID assignment: see GROUP_*_ID constants

		// Determine if this group should be collapsed
		var collapsed bool
		if groupCollapseState != nil {
			// Use the stored collapse state if available
			collapsed = groupCollapseState[id]
		} else {
			// Default: "Other" group is collapsed by default
			collapsed = (id == GROUP_OTHER_ID)
		}

		groupEntries = append(groupEntries, &models.LogEntryView{
			// Create a pseudo LogEntry for the group
			Data: &models.LogEntry{
				ID:        id,
				Text:      name,
				Collapsed: collapsed,
			},
			ViewType: models.LogEntryViewType_Group,
		})
	}

	// For now, put all original entries under the "Other" group
	// Later this can be enhanced to categorize entries into appropriate groups

	mapping := exp.GetMapping()

	normalGroups := groupEntries[:len(groupEntries)-1]
	otherGroup := groupEntries[len(groupEntries)-1]

	for _, entry := range entries {
		mountToGroup(entry, nil, mapping, normalGroups, otherGroup)
	}

	return groupEntries
}
