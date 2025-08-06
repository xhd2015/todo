package search

import (
	"testing"

	"github.com/xhd2015/todo/models"
)

func TestSearchFunctionality(t *testing.T) {
	// Create test entries
	entries := []*models.LogEntryView{
		{
			Data: &models.LogEntry{
				ID:   1,
				Text: "Buy groceries",
			},
			Children: []*models.LogEntryView{
				{
					Data: &models.LogEntry{
						ID:       2,
						Text:     "Buy milk",
						ParentID: 1,
					},
				},
				{
					Data: &models.LogEntry{
						ID:       3,
						Text:     "Buy bread",
						ParentID: 1,
					},
				},
			},
		},
		{
			Data: &models.LogEntry{
				ID:   4,
				Text: "Work on project",
			},
			Children: []*models.LogEntryView{
				{
					Data: &models.LogEntry{
						ID:       5,
						Text:     "Write code",
						ParentID: 4,
					},
				},
			},
		},
		{
			Data: &models.LogEntry{
				ID:   6,
				Text: "Call mom",
			},
		},
	}

	// Test search for "buy"
	filtered := FilterEntriesRecursive(entries, "buy")
	if len(filtered) != 1 {
		t.Errorf("Expected 1 entry for 'buy', got %d", len(filtered))
	}
	if filtered[0].Data.Text != "Buy groceries" {
		t.Errorf("Expected 'Buy groceries', got '%s'", filtered[0].Data.Text)
	}
	if len(filtered[0].Children) != 2 {
		t.Errorf("Expected 2 children for 'Buy groceries', got %d", len(filtered[0].Children))
	}

	// Test search for "milk"
	filtered = FilterEntriesRecursive(entries, "milk")
	if len(filtered) != 1 {
		t.Errorf("Expected 1 entry for 'milk', got %d", len(filtered))
	}
	if filtered[0].Data.Text != "Buy groceries" {
		t.Errorf("Expected parent 'Buy groceries', got '%s'", filtered[0].Data.Text)
	}
	if len(filtered[0].Children) != 1 {
		t.Errorf("Expected 1 child for 'milk' search, got %d", len(filtered[0].Children))
	}
	if filtered[0].Children[0].Data.Text != "Buy milk" {
		t.Errorf("Expected child 'Buy milk', got '%s'", filtered[0].Children[0].Data.Text)
	}

	// Test search for "code"
	filtered = FilterEntriesRecursive(entries, "code")
	if len(filtered) != 1 {
		t.Errorf("Expected 1 entry for 'code', got %d", len(filtered))
	}
	if filtered[0].Data.Text != "Work on project" {
		t.Errorf("Expected parent 'Work on project', got '%s'", filtered[0].Data.Text)
	}

	// Test search for non-existent term
	filtered = FilterEntriesRecursive(entries, "nonexistent")
	if len(filtered) != 0 {
		t.Errorf("Expected 0 entries for 'nonexistent', got %d", len(filtered))
	}

	// Test empty search
	filtered = FilterEntriesRecursive(entries, "")
	if len(filtered) != 3 {
		t.Errorf("Expected 3 entries for empty search, got %d", len(filtered))
	}
}
