package help

import (
	_ "embed"
	"strings"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
)

//go:embed help.md
var helpContent string

// HelpProps contains properties for the Help component
type HelpProps struct {
	ScrollOffset   int // Current scroll position (line offset from top)
	ViewportHeight int // Height of the viewport in lines
}

// Help renders the help page with embedded markdown content
func Help(props HelpProps) *dom.Node {
	lines := strings.Split(helpContent, "\n")

	// Convert content to renderable lines first
	var renderableLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		renderableLines = append(renderableLines, line)
	}

	// Apply viewport scrolling
	totalLines := len(renderableLines)
	startLine := props.ScrollOffset
	endLine := startLine + props.ViewportHeight

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

	var nodes []*dom.Node

	// Add scroll indicator at top if scrolled
	if startLine > 0 {
		nodes = append(nodes, dom.Text("↑ (more content above)", styles.Style{
			Color: colors.GREY_TEXT,
		}))
		nodes = append(nodes, dom.Br())
	}

	// Render visible lines
	visibleLines := renderableLines[startLine:endLine]
	for _, line := range visibleLines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			nodes = append(nodes, dom.Br())
			continue
		}

		// Handle headers
		if strings.HasPrefix(line, "# ") {
			text := strings.TrimPrefix(line, "# ")
			nodes = append(nodes, dom.Text(text, styles.Style{
				Bold:  true,
				Color: colors.GREEN_SUCCESS,
			}))
			nodes = append(nodes, dom.Br())
			continue
		}

		if strings.HasPrefix(line, "## ") {
			text := strings.TrimPrefix(line, "## ")
			nodes = append(nodes, dom.Text(text, styles.Style{
				Bold:  true,
				Color: "cyan",
			}))
			nodes = append(nodes, dom.Br())
			continue
		}

		// Handle list items with code formatting
		if strings.HasPrefix(line, "- ") {
			text := strings.TrimPrefix(line, "- ")

			// Split on " - " to separate command from description
			parts := strings.SplitN(text, " - ", 2)
			if len(parts) == 2 {
				// Format as: command - description
				nodes = append(nodes, dom.Text("  ", styles.Style{}))
				nodes = append(nodes, dom.Text(parts[0], styles.Style{
					Bold:  true,
					Color: "yellow",
				}))
				nodes = append(nodes, dom.Text(" - "+parts[1], styles.Style{
					Color: colors.GREY_TEXT,
				}))
			} else {
				// Regular list item
				nodes = append(nodes, dom.Text("  • "+text, styles.Style{
					Color: colors.GREY_TEXT,
				}))
			}
			nodes = append(nodes, dom.Br())
			continue
		}

		// Regular text
		nodes = append(nodes, dom.Text(line, styles.Style{}))
		nodes = append(nodes, dom.Br())
	}

	// Add scroll indicator at bottom if there's more content
	if endLine < totalLines {
		nodes = append(nodes, dom.Text("↓ (more content below)", styles.Style{
			Color: colors.GREY_TEXT,
		}))
	}

	return dom.Div(dom.DivProps{}, nodes...)
}

// GetTotalLines returns the total number of lines in the help content
func GetTotalLines() int {
	lines := strings.Split(helpContent, "\n")
	return len(lines)
}
