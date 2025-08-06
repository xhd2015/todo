package tree

import (
	"testing"

	"github.com/xhd2015/todo/models"
)

func TestTreeConnectors(t *testing.T) {
	// Create test entries with nested structure
	entries := []*models.LogEntryView{
		{
			Data: &models.LogEntry{
				ID:   1,
				Text: "Parent 1",
			},
			Children: []*models.LogEntryView{
				{
					Data: &models.LogEntry{
						ID:       2,
						Text:     "Child 1.1",
						ParentID: 1,
					},
				},
				{
					Data: &models.LogEntry{
						ID:       3,
						Text:     "Child 1.2",
						ParentID: 1,
					},
					Children: []*models.LogEntryView{
						{
							Data: &models.LogEntry{
								ID:       4,
								Text:     "Grandchild 1.2.1",
								ParentID: 3,
							},
						},
					},
				},
			},
		},
		{
			Data: &models.LogEntry{
				ID:   5,
				Text: "Parent 2",
			},
		},
	}

	// Test the tree structure building logic by simulating what happens in the main app
	type EntryWithDepth struct {
		Entry       *models.LogEntryView
		Index       int
		Depth       int
		IsLastChild []bool
	}

	var flatEntries []EntryWithDepth
	entryIndex := 0

	var addEntryRecursive func(entry *models.LogEntryView, depth int, ancestorIsLast []bool)
	addEntryRecursive = func(entry *models.LogEntryView, depth int, ancestorIsLast []bool) {
		flatEntries = append(flatEntries, EntryWithDepth{
			Entry:       entry,
			Index:       entryIndex,
			Depth:       depth,
			IsLastChild: ancestorIsLast,
		})
		entryIndex++

		// Add children recursively
		for childIndex, child := range entry.Children {
			isLastChild := (childIndex == len(entry.Children)-1)
			// Create ancestor info for child: copy parent's info and add current level
			childAncestorIsLast := make([]bool, depth+1)
			copy(childAncestorIsLast, ancestorIsLast)
			childAncestorIsLast[depth] = isLastChild
			addEntryRecursive(child, depth+1, childAncestorIsLast)
		}
	}

	// Add top-level entries
	topLevelEntries := make([]*models.LogEntryView, 0)
	for _, entry := range entries {
		if entry.Data.ParentID == 0 {
			topLevelEntries = append(topLevelEntries, entry)
		}
	}

	for _, entry := range topLevelEntries {
		addEntryRecursive(entry, 0, []bool{})
	}

	// Verify the structure
	if len(flatEntries) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(flatEntries))
	}

	// Test Parent 1 (depth 0, top-level entry)
	if flatEntries[0].Depth != 0 {
		t.Errorf("Parent 1 should have depth 0, got %d", flatEntries[0].Depth)
	}
	if len(flatEntries[0].IsLastChild) != 0 {
		t.Errorf("Parent 1 should have empty IsLastChild (top-level), got %v", flatEntries[0].IsLastChild)
	}

	// Test Child 1.1 (depth 1, first child of Parent 1)
	if flatEntries[1].Depth != 1 {
		t.Errorf("Child 1.1 should have depth 1, got %d", flatEntries[1].Depth)
	}
	if len(flatEntries[1].IsLastChild) != 1 || flatEntries[1].IsLastChild[0] {
		t.Errorf("Child 1.1 should not be last child, got %v", flatEntries[1].IsLastChild)
	}

	// Test Child 1.2 (depth 1, last child of Parent 1)
	if flatEntries[2].Depth != 1 {
		t.Errorf("Child 1.2 should have depth 1, got %d", flatEntries[2].Depth)
	}
	if len(flatEntries[2].IsLastChild) != 1 || !flatEntries[2].IsLastChild[0] {
		t.Errorf("Child 1.2 should be last child, got %v", flatEntries[2].IsLastChild)
	}

	// Test Grandchild 1.2.1 (depth 2, only child of Child 1.2)
	if flatEntries[3].Depth != 2 {
		t.Errorf("Grandchild 1.2.1 should have depth 2, got %d", flatEntries[3].Depth)
	}
	if len(flatEntries[3].IsLastChild) != 2 || !flatEntries[3].IsLastChild[0] || !flatEntries[3].IsLastChild[1] {
		t.Errorf("Grandchild 1.2.1 should be last child at both levels, got %v", flatEntries[3].IsLastChild)
	}

	// Test Parent 2 (depth 0, top-level entry)
	if flatEntries[4].Depth != 0 {
		t.Errorf("Parent 2 should have depth 0, got %d", flatEntries[4].Depth)
	}
	if len(flatEntries[4].IsLastChild) != 0 {
		t.Errorf("Parent 2 should have empty IsLastChild (top-level), got %v", flatEntries[4].IsLastChild)
	}
}
