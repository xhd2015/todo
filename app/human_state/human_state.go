package human_state

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/component/chart"
	"github.com/xhd2015/todo/component/text"
	"github.com/xhd2015/todo/log"
)

// see https://www.asciiart.eu/image-to-ascii
//
//go:embed man.txt
var manASCII string

const (
	HP_TOTAL_SCORE = 5
	HP_STATE_NAME  = "H/P State"
)

// HumanStateHistoryPoint represents a single history point with date and score
type HumanStateHistoryPoint struct {
	Date  string  // Date in YYYY-MM-DD format
	Score float64 // HP score for that date
}

// HumanState represents the state of human metrics with one hp state having 5 bars
type HumanState struct {
	HpScores        int                                          // 5 bars for the single hp state
	FocusedBarIndex int                                          // Which bar is currently focused (0-4)
	History         []HumanStateHistoryPoint                     // History of HP scores by date
	OnAdjustScore   func(delta int) error                        // Callback for when score is adjusted
	Enqueue         func(action func(ctx context.Context) error) // Async task enqueue function
	LoadStateOnce   func()                                       // Load state once on first access
}

// HumanStatePage renders the complete human state page
func HumanStatePage(humanState *HumanState, onKeyDown func(*dom.DOMEvent)) *dom.Node {
	log.Infof(context.Background(), "DEBUG HumanStatePage: scores=%+v, history=%+v", humanState.HpScores, humanState.History)
	// Get ASCII art as DOM nodes
	asciiText := GetASCIIArt()
	asciiLines := strings.Split(asciiText, "\n")
	var asciiNodes []*dom.Node
	for _, line := range asciiLines {
		asciiNodes = append(asciiNodes, dom.Text(line, styles.Style{Color: colors.GREY_TEXT}))
	}

	// Determine status text based on score
	var statusText string
	if humanState.HpScores > HP_TOTAL_SCORE {
		statusText = "strong"
	} else if humanState.HpScores >= 0 {
		statusText = "tender"
	} else {
		statusText = "weak"
	}

	// Render status text as ASCII art
	statusLines := text.RenderText(statusText, text.TextOptions{})
	var statusNodes []*dom.Node
	for _, line := range statusLines {
		var color string
		if humanState.HpScores > HP_TOTAL_SCORE {
			color = colors.GREEN_SUCCESS
		} else if humanState.HpScores >= 0 {
			color = colors.GREY_TEXT
		} else {
			color = colors.RED_ERROR
		}
		statusNodes = append(statusNodes, dom.Text(line, styles.Style{Color: color, Bold: true}))
	}

	// Create hp bar group as DOM nodes
	hpBarNodes := RenderBars(humanState.HpScores, HP_TOTAL_SCORE, humanState.FocusedBarIndex, func(delta int) {
		humanState.AdjustScore(delta)
	}, func(index int) {
		humanState.FocusedBarIndex = index
	})

	// Render history chart if available
	var chartNodes []*dom.Node
	// Convert history to chart data points
	chartData := make([]chart.DataPoint, len(humanState.History))
	for i, point := range humanState.History {
		chartData[i] = chart.DataPoint{
			X: point.Date,
			Y: point.Score,
		}
	}

	// Render chart lines
	chartLines := chart.RenderLineChartLines(chart.LineChartProps{
		Data:   chartData,
		Width:  100,
		Height: 10,
		Title:  "30-Day History",
	})

	// Convert chart lines to DOM nodes
	for _, line := range chartLines {
		chartNodes = append(chartNodes, dom.Text(line, styles.Style{Color: colors.GREY_TEXT}))
	}

	// Create the main art container
	artNode := dom.HDiv(dom.DivProps{
		Align: dom.AlignBottom,
	},
		dom.Div(dom.DivProps{}, asciiNodes...),
		dom.FixedSpacer(2),
		dom.Div(dom.DivProps{},
			dom.HDiv(dom.DivProps{
				Align: dom.AlignBottom,
			},
				dom.Div(dom.DivProps{}, hpBarNodes...),
				dom.FixedSpacer(2),
				dom.Div(dom.DivProps{}, statusNodes...),
			),
			dom.Div(dom.DivProps{}, chartNodes...),
		),
	)

	// Instructions
	instructions := []string{
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
		dom.Text(""), // Empty line

		// Main content area with ASCII art combined with hp bars
		artNode,

		dom.Text(""), // Empty line

		// Instructions
		dom.HDiv(dom.DivProps{}, instructionNodes...),
	)
}

// AdjustScore increases or decreases the focused bar's score
func (hs *HumanState) AdjustScore(delta int) {
	hs.HpScores += delta
	if hs.OnAdjustScore != nil && hs.Enqueue != nil {
		hs.Enqueue(func(ctx context.Context) error {
			return hs.OnAdjustScore(delta)
		})
	}
}

// GetASCIIArt returns the ASCII art for a male figure
func GetASCIIArt() string {
	return strings.TrimSpace(manASCII)
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
		dom.HDiv(dom.DivProps{},
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

	if hpScores < 0 || hpScores > HP_TOTAL_SCORE {
		color := colors.RED_ERROR
		if hpScores >= 0 {
			color = colors.GREEN_SUCCESS
		}
		nodes = append(nodes, dom.Text(fmt.Sprintf("%d", hpScores), styles.Style{Color: color}))
	}

	return nodes
}
