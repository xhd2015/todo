package tree

func BuildTreePrefix(prefix string, isLast bool) string {
	if prefix == "" {
		return ""
	}

	return prefix + getConnector(isLast)
}

// CalculateChildPrefix calculates the prefix for child entries based on parent's prefix and position
// Returns (prefix, hasVerticalLine)
func CalculateChildPrefix(parentPrefix string, parentIsLast bool, parentEndsWithVertical bool) (string, bool) {
	if parentPrefix == "" {
		// Root level children get 2 spaces
		return "  ", false
	}

	// Non-root level children
	if parentIsLast {
		// Parent was last child, so add appropriate spacing
		if parentEndsWithVertical {
			// If parent prefix has vertical line continuation, add 4 spaces
			return parentPrefix + "    ", false // Still has vertical line from ancestors
		} else {
			// If parent prefix has no vertical line, add 2 spaces
			return parentPrefix + "  ", false
		}
	} else {
		// Parent was not last child, so add vertical bar + space
		return parentPrefix + "│ ", true // Now has vertical line
	}
}

func getConnector(isLast bool) string {
	if isLast {
		return "└─"
	}
	return "├─"
}
