package chart

import (
	"fmt"
	"math"
	"strings"

	"github.com/xhd2015/go-dom-tui/dom"
)

type LineChartProps struct {
	// Data points where X is typically a label (e.g., date) and Y is the value
	Data []DataPoint
	// Width of the chart (default: 60)
	Width int
	// Height of the chart (default: 20)
	Height int
	// Title of the chart
	Title string
}

type DataPoint struct {
	X string  // Label for X axis (e.g., date, category)
	Y float64 // Value for Y axis
}

// RenderLineChartLines renders the line chart and returns it as an array of strings
func RenderLineChartLines(props LineChartProps) []string {
	// Set defaults
	width := props.Width
	if width <= 0 {
		width = 60
	}
	height := props.Height
	if height <= 0 {
		height = 20
	}

	// Find min and max Y values
	minY, maxY := findMinMax(props.Data)

	// Adjust range to nice step intervals
	adjustedMin, adjustedMax, step := adjustRangeToStep(minY, maxY)

	// Build the chart
	var lines []string

	// Add title if provided
	if props.Title != "" {
		lines = append(lines, props.Title)
		lines = append(lines, "")
	}

	// Create the chart grid with adjusted range
	chart := buildChartWithStep(props.Data, width, height, adjustedMin, adjustedMax, step)
	lines = append(lines, chart...)

	// Add X-axis labels
	xLabels := buildXAxisLabels(props.Data, width)
	lines = append(lines, xLabels)

	return lines
}

// LineChart renders a line chart as a dom.Node
// the x is a list of dates, and y is float64 values
// connect the points with lines
func LineChart(props LineChartProps) *dom.Node {
	lines := RenderLineChartLines(props)

	// Convert to dom nodes
	var children []*dom.Node
	for _, line := range lines {
		children = append(children, dom.Text(line))
		children = append(children, dom.Br())
	}

	return dom.Div(dom.DivProps{}, children...)
}

func findMinMax(data []DataPoint) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}

	minY := data[0].Y
	maxY := data[0].Y

	for _, point := range data {
		if point.Y < minY {
			minY = point.Y
		}
		if point.Y > maxY {
			maxY = point.Y
		}
	}

	return minY, maxY
}

// calculateNiceStep calculates a "nice" integer step size for the given range
func calculateNiceStep(dataRange float64, targetSteps int) float64 {
	if dataRange <= 0 {
		return 1
	}

	// Calculate rough step size
	roughStep := dataRange / float64(targetSteps)

	// Find the magnitude (power of 10)
	magnitude := math.Pow(10, math.Floor(math.Log10(roughStep)))

	// Normalize rough step to the magnitude
	normalized := roughStep / magnitude

	// Choose a "nice" step from: 1, 2, 5, 10
	var niceStep float64
	if normalized < 1.5 {
		niceStep = 1
	} else if normalized < 3 {
		niceStep = 2
	} else if normalized < 7 {
		niceStep = 5
	} else {
		niceStep = 10
	}

	return niceStep * magnitude
}

// adjustRangeToStep adjusts min and max values to align with a nice step size
func adjustRangeToStep(minY, maxY float64) (float64, float64, float64) {
	// Handle edge case where all values are the same
	if minY == maxY {
		maxY = minY + 1
	}

	dataRange := maxY - minY

	// Calculate nice step (aim for about 5-10 steps)
	step := calculateNiceStep(dataRange, 8)

	// If the data range is small and consecutive values could differ by 1,
	// force step to be 1 to avoid misleading labels
	// (e.g., -10 and -9 would both round to -10 if step is 2)
	if dataRange <= 20 && step > 1 {
		step = 1
	}

	// Round min down to nearest step
	adjustedMin := math.Floor(minY/step) * step

	// Round max up to nearest step
	adjustedMax := math.Ceil(maxY/step) * step

	// Ensure we have at least one step
	if adjustedMax == adjustedMin {
		adjustedMax = adjustedMin + step
	}

	return adjustedMin, adjustedMax, step
}

func buildChartWithStep(data []DataPoint, width, height int, minY, maxY, step float64) []string {
	// Reserve space for Y-axis labels (8 chars)
	yAxisWidth := 8
	chartWidth := width - yAxisWidth - 3 // -3 for borders and padding
	if chartWidth < 10 {
		chartWidth = 10
	}

	// Step 1: Generate Y-axis tick values from min to max with fixed step
	var yTicks []float64
	for y := minY; y <= maxY; y += step {
		yTicks = append(yTicks, y)
	}
	// Ensure max is included
	if len(yTicks) == 0 || yTicks[len(yTicks)-1] < maxY {
		yTicks = append(yTicks, maxY)
	}

	// Step 2: If there are too many ticks for the height, sample them
	actualHeight := len(yTicks)
	var selectedTicks []float64
	if actualHeight > height {
		// Sample evenly to fit the requested height
		for i := 0; i < height; i++ {
			idx := int(float64(i) * float64(actualHeight-1) / float64(height-1))
			selectedTicks = append(selectedTicks, yTicks[idx])
		}
		actualHeight = height
	} else {
		selectedTicks = yTicks
	}

	// Step 3: Initialize grid with spaces
	lines := make([]string, actualHeight)
	for i := 0; i < actualHeight; i++ {
		lines[i] = strings.Repeat(" ", chartWidth)
	}

	// Helper function to find the chart line index for a Y value
	findLineIndex := func(y float64) int {
		// Find the closest tick
		minDist := math.Abs(y - selectedTicks[0])
		lineIdx := 0
		for i, tick := range selectedTicks {
			dist := math.Abs(y - tick)
			if dist < minDist {
				minDist = dist
				lineIdx = i
			}
		}
		// Convert to chart line index (top to bottom, reversed)
		return actualHeight - 1 - lineIdx
	}

	// Step 4: Plot data points on the determined Y-axis
	for i, point := range data {
		// Calculate X position
		x := int(float64(i) / float64(len(data)-1) * float64(chartWidth-1))
		if len(data) == 1 {
			x = chartWidth / 2
		}

		// Find Y line index
		y := findLineIndex(point.Y)

		// Place marker
		lineRunes := []rune(lines[y])
		if x >= 0 && x < len(lineRunes) {
			lineRunes[x] = '●'
		}
		lines[y] = string(lineRunes)

		// Draw line to next point
		if i < len(data)-1 {
			nextX := int(float64(i+1) / float64(len(data)-1) * float64(chartWidth-1))
			nextY := findLineIndex(data[i+1].Y)

			// Draw line between points
			drawLine(lines, x, y, nextX, nextY, chartWidth)
		}
	}

	// Step 5: Add Y-axis labels
	result := make([]string, actualHeight)
	for i := 0; i < actualHeight; i++ {
		// The Y value for this line
		yValue := selectedTicks[actualHeight-1-i]

		// Format based on whether step is integer or not
		var yLabel string
		if step >= 1 && step == math.Floor(step) {
			yLabel = fmt.Sprintf("%7.0f", yValue)
		} else {
			yLabel = fmt.Sprintf("%7.1f", yValue)
		}

		result[i] = yLabel + " │ " + lines[i]
	}

	return result
}

func buildChart(data []DataPoint, width, height int, minY, maxY float64) []string {
	// Reserve space for Y-axis labels (8 chars)
	yAxisWidth := 8
	chartWidth := width - yAxisWidth - 3 // -3 for borders and padding
	if chartWidth < 10 {
		chartWidth = 10
	}

	lines := make([]string, height)

	// Initialize grid with spaces
	for i := 0; i < height; i++ {
		lines[i] = strings.Repeat(" ", chartWidth)
	}

	// Plot data points
	yRange := maxY - minY
	for i, point := range data {
		// Calculate position
		x := int(float64(i) / float64(len(data)-1) * float64(chartWidth-1))
		if len(data) == 1 {
			x = chartWidth / 2
		}

		// Normalize Y to chart height
		normalizedY := (point.Y - minY) / yRange
		y := height - 1 - int(normalizedY*float64(height-1))

		// Clamp y to valid range
		if y < 0 {
			y = 0
		}
		if y >= height {
			y = height - 1
		}

		// Place marker
		lineRunes := []rune(lines[y])
		if x >= 0 && x < len(lineRunes) {
			lineRunes[x] = '●'
		}
		lines[y] = string(lineRunes)

		// Draw line to next point
		if i < len(data)-1 {
			nextX := int(float64(i+1) / float64(len(data)-1) * float64(chartWidth-1))
			nextNormalizedY := (data[i+1].Y - minY) / yRange
			nextY := height - 1 - int(nextNormalizedY*float64(height-1))

			// Clamp nextY
			if nextY < 0 {
				nextY = 0
			}
			if nextY >= height {
				nextY = height - 1
			}

			// Draw line between points
			drawLine(lines, x, y, nextX, nextY, chartWidth)
		}
	}

	// Add Y-axis labels and borders
	result := make([]string, height)
	for i := 0; i < height; i++ {
		// Calculate Y value for this line
		normalizedY := float64(height-1-i) / float64(height-1)
		yValue := minY + normalizedY*yRange
		yLabel := fmt.Sprintf("%7.1f", yValue)

		result[i] = yLabel + " │ " + lines[i]
	}

	return result
}

func drawLine(lines []string, x1, y1, x2, y2, width int) {
	// Simple line drawing using Bresenham-like algorithm
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)

	sx := 1
	if x1 > x2 {
		sx = -1
	}
	sy := 1
	if y1 > y2 {
		sy = -1
	}

	err := dx - dy
	x, y := x1, y1

	for {
		if x == x2 && y == y2 {
			break
		}

		lineRunes := []rune(lines[y])
		if x >= 0 && x < len(lineRunes) && lineRunes[x] == ' ' {
			// Use different characters for line segments
			if dx > dy {
				lineRunes[x] = '─'
			} else {
				lineRunes[x] = '│'
			}
			lines[y] = string(lineRunes)
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func buildXAxisLabels(data []DataPoint, width int) string {
	yAxisWidth := 8
	chartWidth := width - yAxisWidth - 3
	if chartWidth < 10 {
		chartWidth = 10
	}

	// Show first and last labels
	var label string
	if len(data) > 0 {
		firstLabel := data[0].X
		lastLabel := data[len(data)-1].X

		// Truncate labels if too long
		if len(firstLabel) > 10 {
			firstLabel = firstLabel[:10]
		}
		if len(lastLabel) > 10 {
			lastLabel = lastLabel[:10]
		}

		padding := chartWidth - len(firstLabel) - len(lastLabel)
		if padding < 0 {
			padding = 0
		}

		label = strings.Repeat(" ", yAxisWidth+3) + firstLabel + strings.Repeat(" ", padding) + lastLabel
	}

	return label
}
