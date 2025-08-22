package tree

import (
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
	result := RenderEntriesString(entries)

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
	result := RenderEntriesString(entries)

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

func TestRenderTreeConnectors_ComplexTree(t *testing.T) {
	// Create test entries that replicate the complex tree structure
	// with mock content replacing actual content
	entries := []*models.LogEntryView{
		{
			Data: &models.LogEntry{
				ID:   1,
				Text: "Root Project Optimization",
			},
			Children: []*models.LogEntryView{
				{
					Data: &models.LogEntry{
						ID:       2,
						Text:     "Technical Design Component",
						ParentID: 1,
					},
					Children: []*models.LogEntryView{
						{
							Data: &models.LogEntry{
								ID:       3,
								Text:     "Research Best Practices",
								ParentID: 2,
								Done:     true,
							},
						},
						{
							Data: &models.LogEntry{
								ID:       4,
								Text:     "Final Results Presentation",
								ParentID: 2,
								Done:     true,
							},
						},
						{
							Data: &models.LogEntry{
								ID:       5,
								Text:     "Real World Examples",
								ParentID: 2,
								Done:     true,
							},
						},
						{
							Data: &models.LogEntry{
								ID:       6,
								Text:     "Weekly Planning",
								ParentID: 2,
							},
							Children: []*models.LogEntryView{
								{
									Data: &models.LogEntry{
										ID:       7,
										Text:     "Sprint 8.21",
										ParentID: 6,
									},
									Children: []*models.LogEntryView{
										{
											Data: &models.LogEntry{
												ID:       8,
												Text:     "Graph-based Agent Implementation",
												ParentID: 7,
												Done:     true,
											},
										},
										{
											Data: &models.LogEntry{
												ID:       9,
												Text:     "Legacy Compatibility Update",
												ParentID: 7,
												Done:     true,
											},
											Children: []*models.LogEntryView{
												{
													Data: &models.LogEntry{
														ID:       10,
														Text:     "HTTP Callback Support",
														ParentID: 9,
														Done:     true,
													},
												},
											},
										},
										{
											Data: &models.LogEntry{
												ID:       11,
												Text:     "Proxy Server Implementation",
												ParentID: 7,
												Done:     true,
											},
										},
										{
											Data: &models.LogEntry{
												ID:       12,
												Text:     "Native Tool Integration",
												ParentID: 7,
												Done:     true,
											},
										},
										{
											Data: &models.LogEntry{
												ID:       13,
												Text:     "Mock-based Case Verification",
												ParentID: 7,
											},
										},
										{
											Data: &models.LogEntry{
												ID:       14,
												Text:     "Logs & Trace Integration",
												ParentID: 7,
											},
										},
									},
								},
							},
						},
					},
				},
				{
					Data: &models.LogEntry{
						ID:       15,
						Text:     "Regional Environment Tracing",
						ParentID: 1,
					},
				},
			},
		},
	}

	// Call RenderEntries with the complex tree structure
	result := RenderEntriesString(entries)

	// Expected output with correct tree connectors for the complex structure
	expected := `
• Root Project Optimization
  ├─• Technical Design Component
  │ ├─✓ Research Best Practices
  │ ├─✓ Final Results Presentation
  │ ├─✓ Real World Examples
  │ └─• Weekly Planning
  │     └─• Sprint 8.21
  │       ├─✓ Graph-based Agent Implementation
  │       ├─✓ Legacy Compatibility Update
  │       │ └─✓ HTTP Callback Support
  │       ├─✓ Proxy Server Implementation
  │       ├─✓ Native Tool Integration
  │       ├─• Mock-based Case Verification
  │       └─• Logs & Trace Integration
  └─• Regional Environment Tracing
`

	if diff := assert.Diff(strings.TrimPrefix(expected, "\n"), result); diff != "" {
		t.Error(diff)
	}
}
