package text

import (
	"fmt"
	"strings"
)

// EscapeControlChars safely escapes control characters in the content to prevent terminal issues.
// For ASCII control chars (< 32), it replaces them with escape sequences like \r, \n, \t, etc.
// For non-ASCII control chars, it replaces them with U+xxxx representation.
// Newlines, tabs, and carriage returns are preserved as they're safe for terminal output.
func EscapeControlChars(content string) string {
	var result strings.Builder
	result.Grow(len(content)) // Pre-allocate approximate size

	for _, r := range content {
		// Preserve safe whitespace characters
		if r == '\n' || r == '\t' || r == '\r' {
			result.WriteRune(r)
			continue
		}

		// Check for ASCII control characters (0-31, excluding the safe ones above)
		if r < 32 {
			// Map common control characters to their escape sequences
			switch r {
			case 0:
				result.WriteString("\\0")
			case 7:
				result.WriteString("\\a")
			case 8:
				result.WriteString("\\b")
			case 12:
				result.WriteString("\\f")
			case 11:
				result.WriteString("\\v")
			case 27:
				result.WriteString("\\e")
			default:
				// For other control chars, use hex representation
				result.WriteString(fmt.Sprintf("\\x%02X", r))
			}
			continue
		}

		// Check for DEL character (127)
		if r == 127 {
			result.WriteString("\\x7F")
			continue
		}

		// Check for Unicode control characters (C0 and C1 control codes)
		// C1 control codes: U+0080 to U+009F
		if r >= 0x80 && r <= 0x9F {
			result.WriteString(fmt.Sprintf("U+%04X", r))
			continue
		}

		// Normal character, keep as is
		result.WriteRune(r)
	}

	return result.String()
}
