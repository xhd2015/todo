package run

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/todo/models"
)

// createTestEntry creates a test LogEntryView with the given parameters
func createTestEntry(id int64, text string, parentID int64, done bool) *models.LogEntryView {
	return &models.LogEntryView{
		Data: &models.LogEntry{
			ID:         id,
			Text:       text,
			Done:       done,
			ParentID:   parentID,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		},
		Children: []*models.LogEntryView{},
	}
}

// createDeepNestedStructure creates a deeply nested test structure that triggers compact spacing
func createDeepNestedStructure(maxDepth int) []*models.LogEntryView {
	entries := make([]*models.LogEntryView, 0)

	// Create root entry
	root := createTestEntry(1, "Root Entry", 0, false)
	entries = append(entries, root)

	// Create a linear chain with some siblings to trigger the compact spacing scenario
	currentParent := root
	var currentID int64 = 2

	for depth := 1; depth <= maxDepth; depth++ {
		// Create the main chain child
		mainChild := createTestEntry(currentID, fmt.Sprintf("Child at Depth %d", depth), currentParent.Data.ID, false)
		currentParent.Children = append(currentParent.Children, mainChild)
		currentID++

		// Add a sibling to create the tree structure that needs vertical lines
		if depth < maxDepth {
			sibling := createTestEntry(currentID, fmt.Sprintf("Sibling at Depth %d", depth), currentParent.Data.ID, false)
			currentParent.Children = append(currentParent.Children, sibling)
			currentID++
		}

		// Continue the chain
		currentParent = mainChild
	}

	return entries
}

func TestRenderEntries_DeepNesting(t *testing.T) {
	tests := []struct {
		name   string
		depth  int
		showID bool
	}{
		{
			name:   "Depth 6 - Deep nesting with compact spacing",
			depth:  6,
			showID: false,
		},
		{
			name:   "Depth 7 - Very deep nesting",
			depth:  7,
			showID: false,
		},
		{
			name:   "Depth 8 - Extremely deep nesting",
			depth:  8,
			showID: false,
		},
		{
			name:   "Depth 9 - Ultra deep nesting",
			depth:  9,
			showID: false,
		},
		{
			name:   "Depth 10 - Maximum deep nesting",
			depth:  10,
			showID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test data
			entries := createDeepNestedStructure(tt.depth)

			// Capture output
			output := renderToString(entries, tt.showID, false)

			// Split output into lines
			lines := strings.Split(strings.TrimSpace(output), "\n")

			// Check that we have output
			if len(lines) == 0 {
				t.Fatalf("No output generated")
			}

			// Basic checks
			if !strings.Contains(output, "• Root Entry") {
				t.Errorf("Expected root entry in output, got:\n%s", output)
			}

			// Check that we have the expected depth
			depthText := fmt.Sprintf("Child at Depth %d", tt.depth)
			if !strings.Contains(output, depthText) {
				t.Errorf("Expected depth %d entry in output, got:\n%s", tt.depth, output)
			}

			// Check for tree connectors
			hasTreeConnectors := false
			for _, line := range lines {
				if strings.Contains(line, "├─") || strings.Contains(line, "└─") || strings.Contains(line, "│") {
					hasTreeConnectors = true
					break
				}
			}
			if !hasTreeConnectors {
				t.Errorf("Expected tree connectors in output for depth %d, got:\n%s", tt.depth, output)
			}

			// For debugging, log the actual output structure
			t.Logf("Depth %d output structure:\n%s", tt.depth, output)
		})
	}
}

func TestRenderEntries_WithIDs(t *testing.T) {
	// Test that showID flag works correctly
	entries := createDeepNestedStructure(6)

	output := renderToString(entries, true, false)

	// Check that IDs are shown
	if !strings.Contains(output, "(1)") {
		t.Errorf("Expected ID (1) in output when showID=true, got:\n%s", output)
	}

	if !strings.Contains(output, "(2)") {
		t.Errorf("Expected ID (2) in output when showID=true, got:\n%s", output)
	}
}

func TestRenderEntries_EmptyEntries(t *testing.T) {
	// Test with empty entries list
	output := renderToString([]*models.LogEntryView{}, false, false)

	if strings.TrimSpace(output) != "" {
		t.Errorf("Expected empty output for empty entries, got: %q", output)
	}
}

func TestRenderEntries_SingleEntry(t *testing.T) {
	// Test with single entry
	entries := []*models.LogEntryView{
		createTestEntry(1, "Single Entry", 0, false),
	}

	output := renderToString(entries, false, false)

	expected := "• Single Entry"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected %q in output, got:\n%s", expected, output)
	}
}

func TestRenderEntries_DoneEntries(t *testing.T) {
	// Test with done entries (should show ✓)
	entries := []*models.LogEntryView{
		createTestEntry(1, "Done Entry", 0, true),
	}

	output := renderToString(entries, false, false)

	expected := "✓ Done Entry"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected %q in output, got:\n%s", expected, output)
	}
}
