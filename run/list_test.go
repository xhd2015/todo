package run

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/xgo/support/assert"
)

func TestRenderTreeConnectors(t *testing.T) {
	// Create test entries that replicate the exact structure that was causing
	// tree connector issues, but with renamed content
	entries := []*models.LogEntryView{
		{
			Data: &models.LogEntry{
				ID:   1,
				Text: "Parent Item",
			},
			Children: []*models.LogEntryView{
				{
					Data: &models.LogEntry{
						ID:       2,
						Text:     "Some Content",
						ParentID: 1,
					},
					Children: []*models.LogEntryView{
						{
							Data: &models.LogEntry{
								ID:       3,
								Text:     "First Feature",
								ParentID: 2,
								Done:     true,
							},
							Children: []*models.LogEntryView{
								{
									Data: &models.LogEntry{
										ID:       4,
										Text:     "Change the API implementation",
										ParentID: 3,
									},
								},
								{
									Data: &models.LogEntry{
										ID:       5,
										Text:     "Update the configuration",
										ParentID: 3,
									},
								},
							},
						},
						{
							Data: &models.LogEntry{
								ID:       6,
								Text:     "Second Feature",
								ParentID: 2,
								Done:     true,
							},
							Children: []*models.LogEntryView{
								{
									Data: &models.LogEntry{
										ID:       7,
										Text:     "Fix test case implementation",
										ParentID: 6,
									},
								},
							},
						},
					},
				},
				{
					Data: &models.LogEntry{
						ID:       8,
						Text:     "Another Section",
						ParentID: 1,
					},
				},
			},
		},
	}

	// Call RenderToString with the exact parameters specified
	result := customTenderEntries(entries, false, false)

	// Expected output with correct tree connectors
	expected := `
• Parent Item
  ├─• Some Content
  │ ├─✓ First Feature
  │ │ ├─• Change the API implementation
  │ │ └─• Update the configuration
  │ └─✓ Second Feature
  │     └─• Fix test case implementation
  └─• Another Section
`

	if diff := assert.Diff(strings.TrimPrefix(expected, "\n"), result); diff != "" {
		t.Error(diff)
	}
}

func TestRenderTreeConnectors_Level1_Single(t *testing.T) {
	// Create test entries with single child at level 1 (no "Another Section")
	// This tests the case where "Some Content" is the last/only child
	entries := []*models.LogEntryView{
		{
			Data: &models.LogEntry{
				ID:   1,
				Text: "Parent Item",
			},
			Children: []*models.LogEntryView{
				{
					Data: &models.LogEntry{
						ID:       2,
						Text:     "Some Content",
						ParentID: 1,
					},
					Children: []*models.LogEntryView{
						{
							Data: &models.LogEntry{
								ID:       3,
								Text:     "First Feature",
								ParentID: 2,
								Done:     true,
							},
							Children: []*models.LogEntryView{
								{
									Data: &models.LogEntry{
										ID:       4,
										Text:     "Change the API implementation",
										ParentID: 3,
									},
								},
								{
									Data: &models.LogEntry{
										ID:       5,
										Text:     "Update the configuration",
										ParentID: 3,
									},
								},
							},
						},
						{
							Data: &models.LogEntry{
								ID:       6,
								Text:     "Second Feature",
								ParentID: 2,
								Done:     true,
							},
							Children: []*models.LogEntryView{
								{
									Data: &models.LogEntry{
										ID:       7,
										Text:     "Fix test case implementation",
										ParentID: 6,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Call RenderToString with the exact parameters specified
	result := customTenderEntries(entries, false, false)

	// Expected output with correct tree connectors for single child case
	// Note: "Some Content" uses └─ since it's the only/last child of "Parent Item"
	// But the deeper levels still need vertical connectors since "First Feature" has siblings
	expected := `
• Parent Item
  └─• Some Content
    ├─✓ First Feature
    │ ├─• Change the API implementation
    │ └─• Update the configuration
    └─✓ Second Feature
      └─• Fix test case implementation
`

	if diff := assert.Diff(strings.TrimPrefix(expected, "\n"), result); diff != "" {
		t.Error(diff)
	}
}
func customTenderEntries(entries []*models.LogEntryView, isTTY bool, showID bool) string {
	var b bytes.Buffer
	customTenderEntriesOut(&b, entries, isTTY, showID)
	return b.String()
}

func customTenderEntriesOut(out io.Writer, entries []*models.LogEntryView, isTTY bool, showID bool) {
	// TODO
}
