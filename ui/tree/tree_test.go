package tree

import (
	"testing"
)

func TestBuildTreePrefix(t *testing.T) {
	// Test top-level entry (depth 0)
	prefix := BuildTreePrefix(0, []bool{})
	if prefix != "" {
		t.Errorf("Top-level entry should have empty prefix, got '%s'", prefix)
	}

	// Test first child (depth 1, not last)
	prefix = BuildTreePrefix(1, []bool{false})
	if prefix != "  ├─" {
		t.Errorf("First child should have '  ├─', got '%s'", prefix)
	}

	// Test last child (depth 1, is last)
	prefix = BuildTreePrefix(1, []bool{true})
	if prefix != "  └─" {
		t.Errorf("Last child should have '  └─', got '%s'", prefix)
	}

	// Test grandchild with continuing vertical line
	prefix = BuildTreePrefix(2, []bool{false, false})
	if prefix != "│   ├─" {
		t.Errorf("Grandchild with continuing line should have '│   ├─', got '%s'", prefix)
	}

	// Test grandchild without continuing vertical line
	prefix = BuildTreePrefix(2, []bool{true, false})
	if prefix != "    ├─" {
		t.Errorf("Grandchild without continuing line should have '    ├─', got '%s'", prefix)
	}

	// Test last grandchild with continuing vertical line
	prefix = BuildTreePrefix(2, []bool{false, true})
	if prefix != "│   └─" {
		t.Errorf("Last grandchild with continuing line should have '│   └─', got '%s'", prefix)
	}

	// Test last grandchild without continuing vertical line
	prefix = BuildTreePrefix(2, []bool{true, true})
	if prefix != "    └─" {
		t.Errorf("Last grandchild without continuing line should have '    └─', got '%s'", prefix)
	}
}
