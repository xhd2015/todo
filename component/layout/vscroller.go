package layout

import (
	"context"
	"fmt"

	domLayout "github.com/xhd2015/go-dom-tui/charm/layout"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/log"
)

type VScrollerProps struct {
	Children      []*dom.Node
	Height        int
	BeginIndex    int
	SelectedIndex int // The currently selected/focused item index
}

// VScroller creates a vertical scrolling container that shows a sliding window of children
// starting from BeginIndex and fitting within the specified Height.
// It includes headers at the top and bottom indicating items outside the visible area.
// The SelectedIndex ensures the selected item is always visible.
func VScroller(props VScrollerProps) *dom.Node {
	if len(props.Children) == 0 {
		return dom.Div(dom.DivProps{})
	}

	// Use SliceVertical to calculate the visible range and indicator requirements
	result := SliceVertical(props.Children, props.BeginIndex, props.SelectedIndex, props.Height)

	// Build the result nodes
	var resultNodes []*dom.Node

	// Add top indicator if needed
	if result.ShowTopIndicator {
		topHeader := dom.Div(dom.DivProps{},
			dom.Text(fmt.Sprintf("↑ (%d items above)", result.ItemsAbove), styles.Style{
				Color: "8",
			}),
		)
		resultNodes = append(resultNodes, topHeader)
	}

	// Add visible children
	visibleChildren := props.Children[result.BeginIndex:result.EndIndex]
	resultNodes = append(resultNodes, visibleChildren...)

	// Add bottom indicator if needed
	if result.ShowBottomIndicator {
		bottomHeader := dom.Div(dom.DivProps{},
			dom.Text(fmt.Sprintf("↓ (%d items below)", result.ItemsBelow), styles.Style{
				Color: "8",
			}),
		)
		resultNodes = append(resultNodes, bottomHeader)
	}

	log.Infof(context.Background(), "scroller len(resultNodes): %d", len(resultNodes))

	return dom.Div(dom.DivProps{}, resultNodes...)
}

// SliceVerticalResult contains the result of vertical slicing calculation
type SliceVerticalResult struct {
	BeginIndex          int  // The actual begin index (adjusted if out of bounds)
	EndIndex            int  // The end index (exclusive) for slicing
	ShowTopIndicator    bool // Whether to show "items above" indicator
	ShowBottomIndicator bool // Whether to show "items below" indicator
	ItemsAbove          int  // Number of items above the visible range
	ItemsBelow          int  // Number of items below the visible range
}

// SliceVertical calculates which nodes fit within the given height, accounting for scroll indicators.
// It returns a struct with all the information needed to render the visible portion with indicators.
// The function automatically reserves space for top/bottom indicators when needed.
//
// Parameters:
//   - nodes: The list of all nodes
//   - beginIndex: The starting index for the visible window (scroll position)
//   - selectedIndex: The currently selected item index (must be visible)
//   - height: The total available height
//
// The function ensures both beginIndex and selectedIndex are visible, adjusting beginIndex if necessary.
func SliceVertical(nodes []*dom.Node, beginIndex int, selectedIndex int, height int) SliceVerticalResult {
	// Handle empty nodes
	if len(nodes) == 0 {
		return SliceVerticalResult{
			BeginIndex:          0,
			EndIndex:            0,
			ShowTopIndicator:    false,
			ShowBottomIndicator: false,
			ItemsAbove:          0,
			ItemsBelow:          0,
		}
	}

	// Ensure beginIndex is within bounds
	if beginIndex < 0 {
		beginIndex = 0
	}
	if beginIndex >= len(nodes) {
		beginIndex = len(nodes) - 1
	}

	// Ensure selectedIndex is within bounds
	if selectedIndex < 0 {
		selectedIndex = 0
	}
	if selectedIndex >= len(nodes) {
		selectedIndex = len(nodes) - 1
	}

	// Adjust beginIndex to ensure selectedIndex is visible
	// If selected item is before the current window, scroll up
	if selectedIndex < beginIndex {
		beginIndex = selectedIndex
	}

	// Determine if we need indicators
	hasItemsAbove := beginIndex > 0
	const INDICATOR_HEIGHT = 1

	// Calculate available height for content
	availableHeight := height
	if hasItemsAbove {
		availableHeight -= INDICATOR_HEIGHT
	}

	// Reserve space for bottom indicator (we'll check if we actually need it)
	// We do this conservatively to avoid recalculation
	contentHeight := availableHeight - INDICATOR_HEIGHT
	if contentHeight < 1 {
		contentHeight = 1 // Ensure at least 1 line for content
	}

	// Calculate how many nodes fit in the available content height
	currentHeight := 0
	endIndex := beginIndex

	for i := beginIndex; i < len(nodes); i++ {
		nodeHeight := domLayout.GetNodeRenderedHeight(nodes[i])

		// Check if adding this node would exceed the content height limit
		if currentHeight+nodeHeight > contentHeight {
			break
		}

		currentHeight += nodeHeight
		endIndex = i + 1
	}

	// Check if we actually have items below
	hasItemsBelow := endIndex < len(nodes)

	// If no items below, we can reclaim that reserved space and fit more items
	if !hasItemsBelow && currentHeight < availableHeight {
		// Try to fit more items with the extra space
		for i := endIndex; i < len(nodes); i++ {
			nodeHeight := domLayout.GetNodeRenderedHeight(nodes[i])
			if currentHeight+nodeHeight > availableHeight {
				break
			}
			currentHeight += nodeHeight
			endIndex = i + 1
		}
		// Recheck if we now have items below
		hasItemsBelow = endIndex < len(nodes)
	}

	// If no children fit, show at least the first one (even if it exceeds height)
	if endIndex == beginIndex && beginIndex < len(nodes) {
		endIndex = beginIndex + 1
	}

	// Ensure selectedIndex is visible - if it's beyond endIndex, we need to scroll down
	// This is the key fix: only scroll when selected item goes outside the visible range
	if selectedIndex >= endIndex {
		// Selected item is below the visible range, need to scroll down minimally
		// Start from beginIndex + 1 and increment until selectedIndex becomes visible
		// This ensures minimal scrolling (one item at a time)

		for tryBegin := beginIndex + 1; tryBegin <= selectedIndex; tryBegin++ {
			// Recalculate with this beginIndex
			testHasItemsAbove := tryBegin > 0
			testAvailableHeight := height
			if testHasItemsAbove {
				testAvailableHeight -= INDICATOR_HEIGHT
			}

			testContentHeight := testAvailableHeight - INDICATOR_HEIGHT
			if testContentHeight < 1 {
				testContentHeight = 1
			}

			testCurrentHeight := 0
			testEndIndex := tryBegin

			for i := tryBegin; i < len(nodes); i++ {
				nodeHeight := domLayout.GetNodeRenderedHeight(nodes[i])
				if testCurrentHeight+nodeHeight > testContentHeight {
					break
				}
				testCurrentHeight += nodeHeight
				testEndIndex = i + 1
			}

			// Check if we can fit more with full available height
			testHasItemsBelow := testEndIndex < len(nodes)
			if !testHasItemsBelow && testCurrentHeight < testAvailableHeight {
				for i := testEndIndex; i < len(nodes); i++ {
					nodeHeight := domLayout.GetNodeRenderedHeight(nodes[i])
					if testCurrentHeight+nodeHeight > testAvailableHeight {
						break
					}
					testCurrentHeight += nodeHeight
					testEndIndex = i + 1
				}
			}

			// If selectedIndex is visible in this range, use it and stop
			// This gives us the minimal scroll needed
			if selectedIndex >= tryBegin && selectedIndex < testEndIndex {
				beginIndex = tryBegin
				endIndex = testEndIndex
				hasItemsAbove = testHasItemsAbove
				hasItemsBelow = testEndIndex < len(nodes)
				break
			}
		}
	}

	return SliceVerticalResult{
		BeginIndex:          beginIndex,
		EndIndex:            endIndex,
		ShowTopIndicator:    hasItemsAbove,
		ShowBottomIndicator: hasItemsBelow,
		ItemsAbove:          beginIndex,
		ItemsBelow:          len(nodes) - endIndex,
	}
}
