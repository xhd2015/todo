package human_state

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/log"
)

// see https://www.asciiart.eu/image-to-ascii
//
//go:embed man.txt
var manASCII string

const TOTAL_HP_SCORE = 5

// HumanState represents the state of human metrics with one hp state having 5 bars
type HumanState struct {
	HpScores        int // 5 bars for the single hp state
	FocusedBarIndex int // Which bar is currently focused (0-4)
}

// NewHumanState creates a new human state with default values
func NewHumanState() *HumanState {
	return &HumanState{
		HpScores:        0,
		FocusedBarIndex: -1,
	}
}

// AdjustScore increases or decreases the focused bar's score
func (hs *HumanState) AdjustScore(delta int) {
	hs.HpScores += delta
}

// GetASCIIArt returns the ASCII art for a male figure
func GetASCIIArt() string {
	return strings.TrimSpace(manASCII)
}

// combineAlignBottom combines two DOM node arrays side by side, aligned from the bottom
// Handles cases where len(a) != len(b) by padding the shorter array at the top
func combineAlignBottom(a []*dom.Node, b []*dom.Node, spaceWidth int) []*dom.Node {
	if len(a) == 0 && len(b) == 0 {
		return []*dom.Node{}
	}
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}

	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	result := make([]*dom.Node, maxLen)

	// Calculate padding needed for each array
	aPadding := maxLen - len(a)
	bPadding := maxLen - len(b)

	for i := 0; i < maxLen; i++ {
		var nodeA, nodeB *dom.Node
		var hasA, hasB bool

		// Get node from array a (with top padding)
		if i < aPadding {
			hasA = false
		} else {
			nodeA = a[i-aPadding]
			hasA = true
		}

		// Get node from array b (with top padding)
		if i < bPadding {
			hasB = false
		} else {
			nodeB = b[i-bPadding]
			hasB = true
		}

		// Create div based on what nodes we have
		if hasA && hasB {
			// Both nodes exist - create spacer and combine all three
			spacer := dom.Text(strings.Repeat(" ", spaceWidth), styles.Style{})
			result[i] = dom.Div(dom.DivProps{}, nodeA, spacer, nodeB)
		} else if hasA {
			// Only A exists - just wrap it in a div
			result[i] = dom.Div(dom.DivProps{}, nodeA)
		} else if hasB {
			// Only B exists - just wrap it in a div
			result[i] = dom.Div(dom.DivProps{}, nodeB)
		} else {
			// Neither exists - create empty div
			result[i] = dom.Div(dom.DivProps{})
		}
	}

	return result
}

// RenderBars renders all 5 hp bars as a group of DOM text nodes
func RenderBars(hpScores int, totalScore int, focusedBarIndex int, onAdjustScore func(delta int), onUpdateFocus func(index int)) []*dom.Node {
	var nodes []*dom.Node

	const PLUS_INDEX = -3
	const MINUS_INDEX = -2

	opNode := func(sign string) *dom.Node {
		isMinus := sign == "-"
		focusedIndex := PLUS_INDEX
		scoreDelta := 1
		if isMinus {
			focusedIndex = MINUS_INDEX
			scoreDelta = -2
		}
		var color string
		focused := focusedBarIndex == focusedIndex
		if focused {
			if isMinus {
				color = colors.DARK_RED_1
			} else {
				color = colors.GREEN_SUCCESS
			}
		}

		return dom.TextWithProps(sign, dom.TextNodeProps{
			Focusable: true,
			Focused:   focused,
			OnFocus: func() {
				onUpdateFocus(focusedIndex)
			},
			OnBlur: func() {
				onUpdateFocus(-1)
			},
			OnKeyDown: func(d *dom.DOMEvent) {
				keyEvent := d.KeydownEvent
				switch keyEvent.KeyType {
				case dom.KeyTypeSpace, dom.KeyTypeEnter:
					onAdjustScore(scoreDelta)
				case dom.KeyTypeLeft, dom.KeyTypeRight:
					if isMinus {
						onUpdateFocus(PLUS_INDEX)
					} else {
						onUpdateFocus(MINUS_INDEX)
					}
					d.StopPropagation()
				}
			},
			Style: styles.Style{
				Color: color,
				Bold:  focused,
			},
		})
	}

	// Add label at the top
	nodes = append(nodes,
		dom.Fragment(
			dom.Text("H/P", styles.Style{Bold: true, Color: colors.GREY_TEXT}),
			dom.Text(" "),
			opNode("+"),
			opNode("-"),
		),
	)

	// hpScores >= totalScore
	// . . . . .
	//  totalScore - hpScore

	unscoreNum := totalScore - hpScores

	// Render bars from top to bottom
	for i := 0; i < totalScore; i++ {
		var barLine string

		if i < unscoreNum {
			barLine = "░░░░░░░"
		} else {
			barLine = "███████"
		}

		focused := i == focusedBarIndex

		// Create a single text node for the entire bar row
		// Use focused styling if any bar is focused (we'll handle individual focus differently if needed)
		var style styles.Style
		if focused {
			style.Color = colors.DARK_RED_1
			style.Bold = true
		} else if focusedBarIndex >= 0 && focusedBarIndex < totalScore {
			style.Color = colors.GREEN_SUCCESS
		} else {
			style.Color = colors.GREY_TEXT
		}

		nodes = append(nodes, dom.TextWithProps(barLine, dom.TextNodeProps{
			Focusable: true,
			Focused:   focused,
			OnFocus: func() {
				onUpdateFocus(i)
			},
			OnBlur: func() {
				onUpdateFocus(-1)
			},
			OnKeyDown: func(d *dom.DOMEvent) {
				keyEvent := d.KeydownEvent

				log.Infof(context.Background(), "key event: %s", string(keyEvent.Runes))

				// check +/-
				switch string(keyEvent.Runes) {
				case "+":
					onAdjustScore(1)
				case "-":
					onAdjustScore(-2)
				}
			},
			Style: style,
		}))
	}

	if hpScores < 0 || hpScores > TOTAL_HP_SCORE {
		color := colors.RED_ERROR
		if hpScores >= 0 {
			color = colors.GREEN_SUCCESS
		}
		nodes = append(nodes, dom.Text(fmt.Sprintf("%d", hpScores), styles.Style{Color: color}))
	}

	return nodes
}

// HumanStatePage renders the complete human state page
func HumanStatePage(humanState *HumanState, onKeyDown func(*dom.DOMEvent)) *dom.Node {
	// Get ASCII art as DOM nodes
	asciiText := GetASCIIArt()
	asciiLines := strings.Split(asciiText, "\n")
	var asciiNodes []*dom.Node
	for _, line := range asciiLines {
		asciiNodes = append(asciiNodes, dom.Text(line, styles.Style{Color: colors.GREY_TEXT}))
	}

	// Create hp bar group as DOM nodes
	hpBarNodes := RenderBars(humanState.HpScores, TOTAL_HP_SCORE, humanState.FocusedBarIndex, func(delta int) {
		humanState.AdjustScore(delta)
	}, func(index int) {
		humanState.FocusedBarIndex = index
	})

	// Combine ASCII art with hp bars side by side
	combinedNodes := combineAlignBottom(asciiNodes, hpBarNodes, 2) // 2 spaces between

	// Create the main art container
	artNode := dom.Div(dom.DivProps{}, combinedNodes...)

	// Instructions
	instructions := []string{
		"Human States (HP State)",
		"",
		"Navigation:",
		"↑/↓ - Select bar",
		"+ - Increase score (+1)",
		"- - Decrease score (-2)",
		"ESC - Back to main",
	}

	var instructionNodes []*dom.Node
	for _, instruction := range instructions {
		instructionNodes = append(instructionNodes, dom.Text(instruction, styles.Style{Color: colors.GREY_TEXT}))
	}

	return dom.Div(dom.DivProps{
		OnKeyDown: onKeyDown,
	},
		// Title
		dom.Text("Human States", styles.Style{Bold: true, Color: colors.GREEN_SUCCESS}),
		dom.Text(""), // Empty line

		// Main content area with ASCII art combined with hp bars
		artNode,

		dom.Text(""), // Empty line

		// Instructions
		dom.Div(dom.DivProps{}, instructionNodes...),
	)
}
