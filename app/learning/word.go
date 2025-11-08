package learning

import (
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
)

// WordProps contains properties for rendering a single word
type WordProps struct {
	Text    string
	Focused bool // Whether this word is currently focused
}

// Word renders a single word as an inline span element
// Each word can be individually focused and styled
func Word(props WordProps) *dom.Node {
	wordStyle := styles.Style{}

	if props.Focused {
		wordStyle.Bold = true
		wordStyle.Color = "3" // Yellow for focused word
		wordStyle.Underline = true
	}

	return dom.Span(dom.DivProps{}, dom.Text(props.Text, wordStyle))
}
