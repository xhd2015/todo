package tree

// BuildTreePrefix generates tree connector prefix for hierarchical display
// depth: the depth level of the current item (0 for top-level)
// ancestorIsLast: slice indicating whether each ancestor level is the last child
// Returns a string with appropriate tree connectors (│, ├─, └─) and spacing
func BuildTreePrefix(depth int, ancestorIsLast []bool) string {
	if depth == 0 {
		return ""
	}

	// Build tree connector prefix
	treePrefix := ""

	// Special case for depth 1: always starts with 2 spaces
	if depth == 1 {
		treePrefix = "  "
	} else {
		// For depth >= 2: build prefix by examining each ancestor level
		for d := 0; d < depth-1; d++ {
			if d < len(ancestorIsLast) && !ancestorIsLast[d] {
				// This ancestor has siblings, so we need a vertical line
				if d == 0 {
					// First ancestor (top-level) has siblings: start with 2 spaces + │
					treePrefix += "  │ "
				} else if d == depth-2 {
					// This is the immediate parent level: check if we need compact spacing
					// Use compact spacing only for deep nesting (depth >= 6) to fix alignment issues
					if depth >= 6 {
						treePrefix += "│ "
					} else {
						treePrefix += "  │ "
					}
				} else {
					// This is a middle ancestor level: use consistent spacing
					treePrefix += "  │ "
				}
			} else {
				// This ancestor is the last child, add spacing
				if d == depth-2 {
					// Immediate parent is last child, add extra spacing to align
					treePrefix += "    "
				} else {
					treePrefix += "  "
				}
			}
		}
	}

	// Add the final connector for this item
	if len(ancestorIsLast) >= depth && ancestorIsLast[depth-1] {
		// This is the last child, use └─
		treePrefix += "└─"
	} else {
		// This is not the last child, use ├─
		treePrefix += "├─"
	}

	return treePrefix
}
