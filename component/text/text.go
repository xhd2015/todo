package text

import "strings"

type TextOptions struct {
	Width   int // Width of the text rendering area (for centering)
	Spacing int // Spacing between letters (default: 1)
}

const defaultLetterHeight = 6

// RenderText renders text as large ASCII art by combining individual letters
// Supports: A-Z, 0-9, space, and special characters: - _ + ( )
func RenderText(text string, opts TextOptions) []string {
	text = strings.ToUpper(strings.TrimSpace(text))
	if text == "" {
		return []string{}
	}

	// Set default spacing
	spacing := opts.Spacing
	if spacing == 0 {
		spacing = 1
	}

	// Get all letter renderings
	var letterLines [][]string
	for i := 0; i < len(text); i++ {
		letterLines = append(letterLines, RenderLetter(text[i], opts))
	}

	// Combine letters horizontally
	result := make([]string, defaultLetterHeight)
	for lineIdx := 0; lineIdx < defaultLetterHeight; lineIdx++ {
		var lineBuilder strings.Builder
		for letterIdx, letter := range letterLines {
			if lineIdx < len(letter) {
				lineBuilder.WriteString(letter[lineIdx])
			}
			// Add spacing between letters (but not after the last one)
			if letterIdx < len(letterLines)-1 {
				lineBuilder.WriteString(strings.Repeat(" ", spacing))
			}
		}
		result[lineIdx] = lineBuilder.String()
	}

	return result
}
