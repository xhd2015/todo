package learning

import (
	"fmt"
	"strings"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models"
)

const PAGE_SIZE = 4096 // Each page is 4096 chars

type ReadingProps struct {
	MaterialID    int64
	MaterialTitle string
	CurrentPage   int
	TotalPages    int
	Content       string
	Loading       bool
	Error         string

	// Word navigation
	FocusedWordIndex int
	WordPositions    []models.WordPosition

	// Viewport scrolling
	ScrollOffset   int
	ViewportHeight int

	OnNavigateBack   func()
	OnNextPage       func()
	OnPrevPage       func()
	OnGoToPage       func(page int)
	OnNavigateWord   func(delta int) // Navigate by word (left/right)
	OnNavigateLine   func(delta int) // Navigate by line (up/down)
	OnPageNavigation func(delta int) // Page navigation (h/l keys)
	OnJumpToFirst    func()          // Jump to first word (gg in vim)
	OnJumpToLast     func()          // Jump to last word (G in vim)
	OnKeyG           func()          // Handle 'g' key press for 'gg' sequence
}

func ReadingMaterialPage(props ReadingProps) *dom.Node {
	return dom.Div(dom.DivProps{
		Focusable: true,
		Focused:   true,
		OnKeyDown: func(event *dom.DOMEvent) {
			keyEvent := event.KeydownEvent
			if keyEvent == nil {
				return
			}

			switch keyEvent.KeyType {
			case dom.KeyTypeEsc:
				if props.OnNavigateBack != nil {
					props.OnNavigateBack()
				}
				event.PreventDefault()
			case dom.KeyTypeLeft:
				// Navigate to previous word
				if props.OnNavigateWord != nil {
					props.OnNavigateWord(-1)
				}
				event.PreventDefault()
			case dom.KeyTypeRight:
				// Navigate to next word
				if props.OnNavigateWord != nil {
					props.OnNavigateWord(1)
				}
				event.PreventDefault()
			case dom.KeyTypeUp:
				// Navigate up one line
				if props.OnNavigateLine != nil {
					props.OnNavigateLine(-1)
				}
				event.PreventDefault()
			case dom.KeyTypeDown:
				// Navigate down one line
				if props.OnNavigateLine != nil {
					props.OnNavigateLine(1)
				}
				event.PreventDefault()
			default:
				key := string(keyEvent.Runes)
				switch key {
				case "h":
					// Page navigation with h
					if props.OnPageNavigation != nil {
						props.OnPageNavigation(-1)
					}
					event.PreventDefault()
				case "l":
					// Page navigation with l
					if props.OnPageNavigation != nil {
						props.OnPageNavigation(1)
					}
					event.PreventDefault()
				case "g":
					// Handle 'g' key for 'gg' sequence
					if props.OnKeyG != nil {
						props.OnKeyG()
					}
					event.PreventDefault()
				case "G":
					// Jump to last word (Shift+G in vim)
					if props.OnJumpToLast != nil {
						props.OnJumpToLast()
					}
					event.PreventDefault()
				}
			}
		},
	},
		// Title
		dom.Div(dom.DivProps{},
			dom.Text(props.MaterialTitle, styles.Style{
				Bold: true,
			}),
		),
		// Navigation help
		dom.Div(dom.DivProps{},
			dom.Text("Press ←/→ for word, ↑/↓ for line, h/l for page, gg/G to jump, ESC to go back", styles.Style{
				Color: "8",
			}),
		),
		dom.Div(dom.DivProps{}, dom.Text("")), // Empty line for spacing
		// Content or loading/error state
		func() *dom.Node {
			if props.Loading {
				return dom.Div(dom.DivProps{},
					dom.Text("Loading...", styles.Style{
						Color: "8",
					}),
				)
			}

			if props.Error != "" {
				return dom.Div(dom.DivProps{},
					dom.Text("Error: "+props.Error, styles.Style{
						Color: "1",
					}),
				)
			}

			if props.Content == "" {
				return dom.Div(dom.DivProps{},
					dom.Text("No content available", styles.Style{
						Color: "8",
					}),
				)
			}

			// Render content with word-level highlighting and viewport scrolling
			return renderContentWithWordHighlight(props.Content, props.WordPositions, props.FocusedWordIndex, props.ScrollOffset, props.ViewportHeight)
		}(),
		// Page indicator at bottom
		dom.Div(dom.DivProps{}, dom.Text("")), // Empty line for spacing
		dom.Div(dom.DivProps{},
			dom.Text(fmt.Sprintf("Page %d / %d", props.CurrentPage+1, props.TotalPages), styles.Style{
				Color: "6",
			}),
		),
	)
}

// renderContentWithWordHighlight renders content with the focused word highlighted
// Each word is rendered as a separate inline element for proper focus handling
// Supports viewport scrolling to show only visible lines
func renderContentWithWordHighlight(content string, wordPositions []models.WordPosition, focusedWordIndex int, scrollOffset int, viewportHeight int) *dom.Node {
	if len(wordPositions) == 0 {
		// No word positions, render as plain text
		lines := strings.Split(content, "\n")
		children := make([]*dom.Node, 0, len(lines))
		for _, line := range lines {
			children = append(children, dom.Div(dom.DivProps{},
				dom.Text(line),
			))
		}
		return dom.Div(dom.DivProps{}, children...)
	}

	// Group words by line
	lineWords := make(map[int][]int) // lineIndex -> []wordIndex
	for i, wp := range wordPositions {
		lineWords[wp.LineIndex] = append(lineWords[wp.LineIndex], i)
	}

	// Get all unique line indices and sort them
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Apply viewport scrolling
	startLine := scrollOffset
	endLine := startLine + viewportHeight

	// Clamp bounds
	if startLine < 0 {
		startLine = 0
	}
	if endLine > totalLines {
		endLine = totalLines
	}
	if startLine >= totalLines {
		startLine = totalLines - 1
		if startLine < 0 {
			startLine = 0
		}
	}

	children := make([]*dom.Node, 0)
	linesRendered := 0

	// Add scroll indicator at top if scrolled
	if startLine > 0 {
		children = append(children, dom.Div(dom.DivProps{},
			dom.Text(fmt.Sprintf("↑ (%d lines above)", startLine), styles.Style{
				Color: "8",
			}),
		))
		linesRendered++
	}

	// Render only visible lines
	maxLinesToRender := viewportHeight
	if startLine > 0 {
		maxLinesToRender-- // Account for top scroll indicator
	}
	if endLine < totalLines {
		maxLinesToRender-- // Account for bottom scroll indicator
	}

	actualEndLine := startLine + maxLinesToRender
	if actualEndLine > totalLines {
		actualEndLine = totalLines
	}

	for lineIdx := startLine; lineIdx < actualEndLine; lineIdx++ {
		line := lines[lineIdx]
		wordIndices, hasWords := lineWords[lineIdx]
		if !hasWords || len(line) == 0 {
			// No words on this line, render as plain text
			children = append(children, dom.Div(dom.DivProps{},
				dom.Text(line),
			))
			continue
		}

		// Render line with each word as a separate inline element
		lineChildren := make([]*dom.Node, 0)
		lastPos := 0

		for _, wordIdx := range wordIndices {
			wp := wordPositions[wordIdx]

			// Calculate position within the line
			lineStartInContent := 0
			for i := 0; i < lineIdx; i++ {
				lineStartInContent += len(lines[i]) + 1 // +1 for newline
			}

			wordStartInLine := wp.StartPos - lineStartInContent
			wordEndInLine := wp.EndPos - lineStartInContent

			// Add text before the word (spaces, punctuation, etc.)
			if wordStartInLine > lastPos {
				beforeText := line[lastPos:wordStartInLine]
				lineChildren = append(lineChildren, dom.Text(beforeText))
			}

			// Render each word using the Word component
			if wordEndInLine <= len(line) {
				wordText := line[wordStartInLine:wordEndInLine]
				isFocused := wordIdx == focusedWordIndex
				lineChildren = append(lineChildren, Word(WordProps{
					Text:    wordText,
					Focused: isFocused,
				}))
				lastPos = wordEndInLine
			}
		}

		// Add remaining text after last word
		if lastPos < len(line) {
			lineChildren = append(lineChildren, dom.Text(line[lastPos:]))
		}

		children = append(children, dom.Div(dom.DivProps{}, lineChildren...))
		linesRendered++
	}

	// Add scroll indicator at bottom if there's more content
	if endLine < totalLines {
		children = append(children, dom.Div(dom.DivProps{},
			dom.Text(fmt.Sprintf("↓ (%d lines below)", totalLines-endLine), styles.Style{
				Color: "8",
			}),
		))
		linesRendered++
	}

	// Pad with empty lines to ensure consistent viewport height
	for linesRendered < viewportHeight {
		children = append(children, dom.Div(dom.DivProps{}, dom.Text("")))
		linesRendered++
	}

	return dom.Div(dom.DivProps{}, children...)
}
