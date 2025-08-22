package tree

import "strings"

func BuildTreePrefix(prefix string, isLast bool) string {
	if prefix == "" {
		return ""
	}

	return prefix + getConnector(isLast)
}

// CalculateChildPrefix calculates the prefix for child entries based on parent's prefix and position
func CalculateChildPrefix(parentPrefix string, parentIsLast bool) string {
	if parentPrefix == "" {
		// Root level children get 2 spaces
		return "  "
	}

	// Non-root level children
	if parentIsLast {
		// Parent was last child, so add appropriate spacing
		if strings.HasSuffix(parentPrefix, "│ ") {
			// If prefix ends with │ (vertical bar + space), add 4 spaces
			return parentPrefix + "    "
		} else {
			// If prefix ends with just spaces, add 2 spaces
			return parentPrefix + "  "
		}
	} else {
		// Parent was not last child, so add vertical bar + space
		return parentPrefix + "│ "
	}
}

func getConnector(isLast bool) string {
	if isLast {
		return "└─"
	}
	return "├─"
}
