package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xhd2015/go-dom-tui/charm/renderer"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/todo/models"
)

// createTestEntries creates test entries for pagination testing
func createTestEntries(count int) models.LogEntryViews {
	entries := make(models.LogEntryViews, count)
	for i := 0; i < count; i++ {
		entries[i] = &models.LogEntryView{
			Data: &models.LogEntry{
				ID:       int64(i + 1),
				Text:     fmt.Sprintf("Todo item %d", i+1),
				ParentID: 0,
				Done:     false,
			},
			Children: nil,
		}
	}
	return entries
}

// createTestState creates a test state with given entries and pagination settings
func createTestState(entries models.LogEntryViews, sliceStart int, selectedID int64, maxEntries int) *State {
	return &State{
		Entries:    entries,
		SliceStart: sliceStart,
		SelectedEntry: EntryIdentity{
			EntryType: TreeEntryType_Log,
			ID:        selectedID,
		},
	}
}

func TestRenderEntryTree_NoPagination(t *testing.T) {
	// Test with fewer entries than MaxEntries (should not show indicators)
	entries := createTestEntries(5)
	maxEntries := 30
	state := createTestState(entries, 0, 0, maxEntries)

	props := TreeProps{
		State:   state,
		Entries: []TreeEntry{},
	}

	nodes := Tree(props)

	// Create a ul node to contain the rendered entries
	ulNode := dom.Ul(dom.DivProps{}, nodes...)

	// Render using InteractiveCharmRenderer
	renderer := renderer.NewInteractiveCharmRenderer()
	output := renderer.Render(ulNode)

	// Should not contain pagination indicators
	if strings.Contains(output, "more above") {
		t.Error("Should not show 'more above' indicator when no pagination needed")
	}
	if strings.Contains(output, "more below") {
		t.Error("Should not show 'more below' indicator when no pagination needed")
	}

	// Should contain all 5 todo items
	for i := 1; i <= 5; i++ {
		expected := fmt.Sprintf("Todo item %d", i)
		if !strings.Contains(output, expected) {
			t.Errorf("Expected to find '%s' in output", expected)
		}
	}
}

func TestRenderEntryTree_ShowDownIndicator(t *testing.T) {
	// Test with more entries than MaxEntries, starting from beginning
	entries := createTestEntries(35) // More than MaxEntries (30)
	state := createTestState(entries, 0, 0, 30)

	props := TreeProps{
		State:        state,
		Entries:      []TreeEntry{},
		EntriesAbove: 0,
		EntriesBelow: 0,
	}

	nodes := Tree(props)
	ulNode := dom.Ul(dom.DivProps{}, nodes...)

	renderer := renderer.NewInteractiveCharmRenderer()
	output := renderer.Render(ulNode)

	// Should not show up indicator (we're at the beginning)
	if strings.Contains(output, "more above") {
		t.Error("Should not show 'more above' indicator when at beginning")
	}

	// Should show down indicator with correct count (35 - 30 = 5)
	expectedDown := "↓ 5 more below"
	if !strings.Contains(output, expectedDown) {
		t.Errorf("Expected to find '%s' in output, got: %s", expectedDown, output)
	}

	// Should contain first 30 items
	for i := 1; i <= 30; i++ {
		expected := fmt.Sprintf("Todo item %d", i)
		if !strings.Contains(output, expected) {
			t.Errorf("Expected to find '%s' in output", expected)
		}
	}

	// Should NOT contain items beyond 30
	for i := 31; i <= 35; i++ {
		unexpected := fmt.Sprintf("Todo item %d", i)
		if strings.Contains(output, unexpected) {
			t.Errorf("Should not find '%s' in output when paginated", unexpected)
		}
	}
}

func TestRenderEntryTree_ShowUpIndicator(t *testing.T) {
	// Test with more entries than MaxEntries, starting from middle
	entries := createTestEntries(35)            // More than MaxEntries (30)
	state := createTestState(entries, 5, 0, 30) // Start from entry 6

	props := TreeProps{
		State:        state,
		Entries:      []TreeEntry{},
		EntriesAbove: 5,
		EntriesBelow: 0,
	}

	nodes := Tree(props)
	ulNode := dom.Ul(dom.DivProps{}, nodes...)

	renderer := renderer.NewInteractiveCharmRenderer()
	output := renderer.Render(ulNode)

	// Should show up indicator with correct count (5 entries above)
	expectedUp := "↑ 5 more above"
	if !strings.Contains(output, expectedUp) {
		t.Errorf("Expected to find '%s' in output, got: %s", expectedUp, output)
	}

	// Should not show down indicator (5 + 30 = 35, which is all entries)
	if strings.Contains(output, "more below") {
		t.Error("Should not show 'more below' indicator when showing last entries")
	}

	// Should contain items 6-35 (entries[5:35])
	for i := 6; i <= 35; i++ {
		expected := fmt.Sprintf("Todo item %d", i) // Just check for the text, not the bullet
		if !strings.Contains(output, expected) {
			t.Errorf("Expected to find '%s' in output", expected)
		}
	}

	// Should NOT contain first 5 items (use word boundaries to avoid false matches)
	for i := 1; i <= 5; i++ {
		unexpected := fmt.Sprintf("•Todo item %d\n", i) // More specific pattern
		if strings.Contains(output, unexpected) {
			t.Errorf("Should not find '%s' in output when paginated. Full output:\n%s", unexpected, output)
		}
	}
}

func TestRenderEntryTree_ShowBothIndicators(t *testing.T) {
	// Test with more entries than MaxEntries, starting from middle
	entries := createTestEntries(50)             // Much more than MaxEntries (30)
	state := createTestState(entries, 10, 0, 30) // Start from entry 11

	props := TreeProps{
		State:        state,
		Entries:      []TreeEntry{},
		EntriesAbove: 10,
		EntriesBelow: 10,
	}

	nodes := Tree(props)
	ulNode := dom.Ul(dom.DivProps{}, nodes...)

	renderer := renderer.NewInteractiveCharmRenderer()
	output := renderer.Render(ulNode)

	// Should show up indicator with correct count (10 entries above)
	expectedUp := "↑ 10 more above"
	if !strings.Contains(output, expectedUp) {
		t.Errorf("Expected to find '%s' in output, got: %s", expectedUp, output)
	}

	// Should show down indicator with correct count (50 - 40 = 10 entries below)
	expectedDown := "↓ 10 more below"
	if !strings.Contains(output, expectedDown) {
		t.Errorf("Expected to find '%s' in output, got: %s", expectedDown, output)
	}

	// Should contain items 11-40 (entries[10:40])
	for i := 11; i <= 40; i++ {
		expected := fmt.Sprintf("Todo item %d", i) // Just check for the text, not the bullet
		if !strings.Contains(output, expected) {
			t.Errorf("Expected to find '%s' in output", expected)
		}
	}

	// Should NOT contain first 10 items (use word boundaries to avoid false matches)
	for i := 1; i <= 10; i++ {
		unexpected := fmt.Sprintf("•Todo item %d\n", i) // More specific pattern
		if strings.Contains(output, unexpected) {
			t.Errorf("Should not find '%s' in output when paginated. Full output:\n%s", unexpected, output)
		}
	}

	// Should NOT contain last 10 items (use word boundaries to avoid false matches)
	for i := 41; i <= 50; i++ {
		unexpected := fmt.Sprintf("•Todo item %d\n", i) // More specific pattern
		if strings.Contains(output, unexpected) {
			t.Errorf("Should not find '%s' in output when paginated", unexpected)
		}
	}
}

func TestRenderEntryTree_EdgeCaseNavigation(t *testing.T) {
	// Test edge cases for navigation boundaries
	entries := createTestEntries(35)

	// Test case 1: SliceStart at end boundary
	state := createTestState(entries, 35, 0, 30) // Beyond total entries
	props := TreeProps{
		State:        state,
		Entries:      []TreeEntry{},
		EntriesAbove: 5, // This should be adjusted to 5 (35 - 30)
		EntriesBelow: 0,
	}

	nodes := Tree(props)
	ulNode := dom.Ul(dom.DivProps{}, nodes...)

	renderer := renderer.NewInteractiveCharmRenderer()
	output := renderer.Render(ulNode)

	// Should show up indicator (5 entries above)
	expectedUp := "↑ 5 more above"
	if !strings.Contains(output, expectedUp) {
		t.Errorf("Expected to find '%s' in output for edge case, got: %s", expectedUp, output)
	}

	// Test case 2: Negative SliceStart
	state2 := createTestState(entries, -5, 0, 30) // Negative, should be adjusted to 0
	props2 := TreeProps{
		State:        state2,
		Entries:      []TreeEntry{},
		EntriesAbove: 0,
		EntriesBelow: 5,
	}

	nodes2 := Tree(props2)
	ulNode2 := dom.Ul(dom.DivProps{}, nodes2...)

	output2 := renderer.Render(ulNode2)

	// Should not show up indicator (adjusted to start from 0)
	if strings.Contains(output2, "more above") {
		t.Error("Should not show 'more above' indicator when SliceStart adjusted to 0")
	}

	// Should show down indicator
	expectedDown2 := "↓ 5 more below"
	if !strings.Contains(output2, expectedDown2) {
		t.Errorf("Expected to find '%s' in output for negative edge case", expectedDown2)
	}
}
