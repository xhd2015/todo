package tree

// BuildTreePrefix generates tree connector prefix for hierarchical display
// depth: the depth level of the current item (0 for top-level)
// ancestorIsLast: slice indicating whether each ancestor level is the last child
// Returns a string with appropriate tree connectors (│, ├─, └─) and spacing
func BuildTreePrefix(depth int, ancestorIsLast []bool) string {
	// Build tree connector prefix
	treePrefix := ""
	for d := 0; d < depth; d++ {
		// Check if ancestor at level d has more siblings
		// Only draw vertical line if the ancestor is not at the top level and has more siblings
		if d < len(ancestorIsLast) && !ancestorIsLast[d] && d < depth-1 {
			// There are more siblings at this ancestor level, so draw a vertical line
			treePrefix += "│ "
		} else {
			// No more siblings at this ancestor level, so just add spacing
			treePrefix += "  "
		}
	}

	// Add the final connector for this item
	if depth > 0 {
		if len(ancestorIsLast) >= depth && ancestorIsLast[depth-1] {
			// This is the last child, use └─
			treePrefix += "└─"
		} else {
			// This is not the last child, use ├─
			treePrefix += "├─"
		}
	}

	return treePrefix
}
