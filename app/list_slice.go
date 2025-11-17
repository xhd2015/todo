package app

import (
	"github.com/xhd2015/todo/app/exp"
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/models/states"
	"github.com/xhd2015/todo/ui/search"
	"github.com/xhd2015/todo/ui/tree"
)

// _MountGroupInfo tracks the group association and staging children for an entry during group organization
type _MountGroupInfo struct {
	LogGroupID      int64
	StagingChildren models.LogEntryViews
}

// mountToGroup mounts a log entry to its appropriate group based on mapping and parent chain
// Returns a cloned entry with proper group association and children
func mountToGroup(entry *models.LogEntryView, parents []*_MountGroupInfo, groupMapping map[int64]int64, normalGroups models.LogEntryViews, otherGroup *models.LogEntryView) *models.LogEntryView {
	normalGroupID := groupMapping[entry.Data.ID]
	var foundNormalGroupID int64
	var foundNormalGroup *models.LogEntryView
	if normalGroupID != 0 {
		for _, groupEntry := range normalGroups {
			if groupEntry.Data.ID == normalGroupID {
				foundNormalGroupID = normalGroupID
				foundNormalGroup = groupEntry
				break
			}
		}
	}

	n := len(parents)

	var targetGroup *models.LogEntryView

	var attachToParent *_MountGroupInfo
	if foundNormalGroupID != 0 {
		for i := n - 1; i >= 0; i-- {
			p := parents[i]
			if p.LogGroupID == foundNormalGroupID {
				attachToParent = p
				break
			}
		}
		if attachToParent == nil {
			targetGroup = foundNormalGroup
		}
	} else {
		// follow nearest parent
		if n > 0 {
			attachToParent = parents[n-1]
		} else {
			targetGroup = otherGroup
		}
	}

	var logAndGroup *_MountGroupInfo

	cloneEntry := *entry
	if targetGroup != nil {
		targetGroup.Children = append(targetGroup.Children, &cloneEntry)
		logAndGroup = &_MountGroupInfo{
			LogGroupID: targetGroup.Data.ID,
		}
	} else {
		if attachToParent == nil {
			panic("attach to parent should not be nil when target group is nil")
		}
		attachToParent.StagingChildren = append(attachToParent.StagingChildren, &cloneEntry)
		logAndGroup = &_MountGroupInfo{
			LogGroupID: attachToParent.LogGroupID,
		}
	}
	parents = append(parents, logAndGroup)

	for _, child := range entry.Children {
		mountToGroup(child, parents, groupMapping, normalGroups, otherGroup)
	}

	cloneEntry.Children = logAndGroup.StagingChildren
	return &cloneEntry
}

type EntryOptions struct {
	// MaxEntries int
	// SliceStart         int
	SelectedID         models.EntryIdentity
	SearchSelectedID   models.EntryIdentity
	SelectedSource     states.SelectedSource
	ZenMode            bool
	SearchActive       bool
	Search             string
	ShowNotes          bool
	FocusingEntryID    models.EntryIdentity
	ExpandAll          bool
	ViewMode           states.ViewMode
	GroupCollapseState map[int64]bool
}

func flattenEntryTree(entries models.LogEntryViews, opts EntryOptions) []states.TreeEntry {
	var focusedRootPath []string
	// Process focusing if focusingEntryID is provided
	if opts.FocusingEntryID.IsSet() {
		entries, focusedRootPath = processFocusedEntries(entries, opts.FocusingEntryID)
	}

	// Organize entries into groups if ViewMode is Group
	if opts.ViewMode == states.ViewMode_Group {
		entries = organizeEntriesIntoGroups(entries, opts.GroupCollapseState)
	}

	// Filter entries based on search query if active
	entriesToRender := applyFilter(entries, opts.ZenMode, opts.SearchActive, opts.Search)

	// Process collapsed entries to hide children and add count information
	if !opts.ExpandAll {
		entriesToRender, _ = processCollapsedEntries(entriesToRender, false, opts.SelectedID, opts.SearchSelectedID, opts.SearchActive, opts.ZenMode)
	}

	// Add top-level entries (ParentID == 0)
	topLevelEntries := make([]*models.LogEntryView, 0)
	for _, entry := range entriesToRender {
		if entry.Data.ParentID == 0 {
			topLevelEntries = append(topLevelEntries, entry)
		}
	}

	var flatEntries []states.TreeEntry

	// Add focused root path as the first entry if in focused mode
	if len(focusedRootPath) > 0 {
		focusedTreeEntry := states.TreeEntry{
			Type:   models.LogEntryViewType_FocusedItem,
			Prefix: "",
			IsLast: false,
			FocusedItem: &states.TreeFocusedItem{
				RootPath: focusedRootPath,
			},
		}
		flatEntries = append(flatEntries, focusedTreeEntry)
	}

	for i, entry := range topLevelEntries {
		isLast := i == len(topLevelEntries)-1
		flatEntries = addEntryRecursive(flatEntries, entry, 0, "", isLast, false, opts.ShowNotes)
	}
	return flatEntries
}

// organizeEntriesIntoGroups organizes entries into pseudo groups based on their properties
func organizeEntriesIntoGroups(entries models.LogEntryViews, groupCollapseState map[int64]bool) models.LogEntryViews {
	// Create group entries in order: Deadline, WorkPerf, LifeEnhance, WorkHack, LifeHack, Other
	groupNames := []string{"Deadline", "WorkPerf", "LifeEnhance", "WorkHack", "LifeHack", "Other"}
	groupEntries := make(models.LogEntryViews, 0, len(groupNames))

	for i, name := range groupNames {
		id := int64(i + 1) // Natural ID assignment: see GROUP_*_ID constants

		// Determine if this group should be collapsed
		collapsed, ok := groupCollapseState[id]
		if !ok {
			// Default: "Other" group is collapsed by default
			collapsed = (id == states.GROUP_OTHER_ID)
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

	mapping := exp.GetMapping()

	normalGroups := groupEntries[:len(groupEntries)-1]
	otherGroup := groupEntries[len(groupEntries)-1]

	// Use optimized queue-based approach instead of recursive mountToGroup
	for _, entry := range entries {
		mountToGroup(entry, nil, mapping, normalGroups, otherGroup)
	}

	return groupEntries
}

func addEntryRecursive(flatEntries []states.TreeEntry, entry *models.LogEntryView, depth int, prefix string, isLast bool, hasVerticalLine bool, globalShowNotes bool) []states.TreeEntry {
	// Implement 'v implies n': if IncludeHistory is true, also show notes
	// Also show notes if global notes mode is enabled
	showNotes := entry.IncludeNotes || entry.IncludeHistory || globalShowNotes
	return addEntryRecursiveWithHistory(flatEntries, entry, depth, prefix, isLast, hasVerticalLine, entry.IncludeHistory, showNotes, globalShowNotes)
}

func addEntryRecursiveWithHistory(flatEntries []states.TreeEntry, entry *models.LogEntryView, depth int, prefix string, isLast bool, hasVerticalLine bool, showNotesInSubtree bool, showNotesFromParent bool, globalShowNotes bool) []states.TreeEntry {
	// Determine entry type based on the ViewType field
	var treeEntry states.TreeEntry

	if entry.ViewType == models.LogEntryViewType_Group {
		treeEntry = states.TreeEntry{
			Type:   models.LogEntryViewType_Group,
			Prefix: prefix,
			IsLast: isLast,
			Entry:  entry,
			Group: &states.TreeGroup{
				ID:   entry.Data.ID,
				Name: entry.Data.Text,
			},
		}
	} else {
		treeEntry = states.TreeEntry{
			Type:   models.LogEntryViewType_Log,
			Prefix: prefix,
			IsLast: isLast,
			Entry:  entry,
			Log:    &states.TreeLog{},
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
			flatEntries = append(flatEntries, states.TreeEntry{
				Type: models.LogEntryViewType_Note,
				Entry: &models.LogEntryView{
					Data: &models.LogEntry{
						ID:   note.Data.ID,
						Text: "📝 " + note.Data.Text,
					},
					ViewType:       models.LogEntryViewType_Note,
					EntryIDForNote: entry.Data.ID,
				},
				Prefix: notePrefix,
				IsLast: isLastNote,
				Note: &states.TreeNote{
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

func applyFilter(list models.LogEntryViews, zenMode bool, searchActive bool, searchQuery string) models.LogEntryViews {
	// Filter entries based on search query if active
	entriesToRender := list

	if zenMode {
		entriesToRender = search.FilterEntries(list, func(entry *models.LogEntryView) bool {
			return entry.Data.HighlightLevel > 0 && !entry.Data.Done
		})
	}

	if searchActive {
		// when searchQuery is empty, it means clear search labels
		entriesToRender = search.FilterEntriesQuery(entriesToRender, searchQuery)
	}

	return entriesToRender
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
// The selectedID path is kept visible even if parents are collapsed
func processCollapsedEntries(entries models.LogEntryViews, anyParentCollapsed bool, selectedID models.EntryIdentity, searchSelectedID models.EntryIdentity, searchActive bool, zenMode bool) (models.LogEntryViews, models.LogEntryViews) {
	showEntries := make(models.LogEntryViews, 0, len(entries))
	collapsedEntries := make(models.LogEntryViews, 0, len(entries))
	for _, entry := range entries {
		cloned := *entry
		clonedEntry := &cloned

		// clonedEntry.Children
		shouldCollapse := anyParentCollapsed || clonedEntry.Data.Collapsed
		shownChildren, collapsedChildren := processCollapsedEntries(clonedEntry.Children, shouldCollapse, selectedID, searchSelectedID, searchActive, zenMode)

		clonedEntry.Children = shownChildren
		clonedEntry.CollapsedChildren = collapsedChildren
		clonedEntry.CollapsedCount = countAllChildren(collapsedChildren)

		if clonedEntry.Data.Collapsed && len(shownChildren) > 0 {
			cloneData := *clonedEntry.Data
			clonedEntry.Data = &cloneData
		}

		addToShow := true
		// then select from children
		if anyParentCollapsed {
			addToShow = false
			if len(shownChildren) > 0 || shouldShowEvenIfCollapsed(clonedEntry, selectedID, searchSelectedID, searchActive, zenMode) {
				addToShow = true
			}
		}
		if addToShow {
			showEntries = append(showEntries, clonedEntry)
		} else {
			collapsedEntries = append(collapsedEntries, clonedEntry)
		}
	}
	return showEntries, collapsedEntries
}

func shouldShowEvenIfCollapsed(entry *models.LogEntryView, selectedID models.EntryIdentity, searchSelectedID models.EntryIdentity, searchActive bool, zenMode bool) bool {
	if selectedID.IsSet() && entry.SameIdentity(selectedID) {
		return true
	}
	if searchSelectedID.IsSet() && entry.SameIdentity(searchSelectedID) {
		return true
	}
	if searchActive && isSearchMatchEntry(entry) {
		return true
	}
	if zenMode && isZenModeEntry(entry) {
		return true
	}
	return false
}

func isZenModeEntry(entry *models.LogEntryView) bool {
	return entry.Data.HighlightLevel > 0 && !entry.Data.Done
}

func isSearchMatchEntry(entry *models.LogEntryView) bool {
	return len(entry.MatchTexts) > 0
}

// countAllChildren recursively counts all children and their descendants
func countAllChildren(children []*models.LogEntryView) int {
	count := len(children)
	for _, child := range children {
		count += child.CollapsedCount
	}
	return count
}
