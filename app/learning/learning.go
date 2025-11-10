package learning

import (
	"fmt"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/component/layout"
	"github.com/xhd2015/todo/models"
)

type LearningMaterialListProps struct {
	Materials         []*models.LearningMaterial
	SelectedIndex     int
	ScrollOffset      int
	ContainerHeight   int
	ContainerWidth    int
	OnNavigateBack    func()
	OnReload          func()
	OnSelectMaterial  func(index int)
	OnOpenMaterial    func(materialID int64)
	OnNavigateUp      func()
	OnNavigateDown    func()
	OnUpdateScrollPos func(scrollOffset int)
}

func LearningMaterialList(props LearningMaterialListProps) *dom.Node {
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
					event.StopPropagation()
				}
			case dom.KeyTypeUp:
				if props.OnNavigateUp != nil {
					props.OnNavigateUp()
				}
			case dom.KeyTypeDown:
				if props.OnNavigateDown != nil {
					props.OnNavigateDown()
				}
				event.PreventDefault()
			case dom.KeyTypeEnter:
				if props.OnOpenMaterial != nil && len(props.Materials) > 0 && props.SelectedIndex >= 0 && props.SelectedIndex < len(props.Materials) {
					props.OnOpenMaterial(props.Materials[props.SelectedIndex].ID)
				}
				event.PreventDefault()
			default:
				key := string(keyEvent.Runes)
				switch key {
				case "r", "R":
					if props.OnReload != nil {
						props.OnReload()
					}
					event.PreventDefault()
				}
			}
		},
	},
		func() *dom.Node {
			// Fixed header lines
			const HEADER_LINES = 3 // Title + Help + Empty line
			availableHeight := props.ContainerHeight - HEADER_LINES

			// Ensure minimum height
			if availableHeight < 5 {
				availableHeight = 5
			}

			// Build header nodes
			headerNodes := []*dom.Node{
				dom.Div(dom.DivProps{},
					dom.Text("Learning Materials (Last 10)", styles.Style{
						Bold: true,
					}),
				),
				dom.Div(dom.DivProps{},
					dom.Text("Press ↑/↓ to navigate, Enter to read, 'r' to reload, ESC to go back", styles.Style{
						Color: colors.TextSecondary,
					}),
				),
				dom.Div(dom.DivProps{}, dom.Text("")), // Empty line for spacing
			}

			// Handle empty materials case
			if len(props.Materials) == 0 {
				contentNode := dom.Div(dom.DivProps{},
					dom.Text("No learning materials found", styles.Style{
						Color: colors.TextSecondary,
					}),
				)
				return dom.Fragment(append(headerNodes, contentNode)...)
			}

			// Build all material item nodes
			allItemNodes := make([]*dom.Node, 0, len(props.Materials))
			for i, material := range props.Materials {
				isSelected := i == props.SelectedIndex
				allItemNodes = append(allItemNodes, renderMaterialItem(i+1, material, isSelected))
			}

			// Use VScroller to handle the scrolling logic and indicators
			// VScroller will automatically calculate visible items and show indicators
			scrollerNode := layout.VScroller(layout.VScrollerProps{
				Children:      allItemNodes,
				Height:        availableHeight,
				BeginIndex:    props.ScrollOffset,
				SelectedIndex: props.SelectedIndex,
			})

			// Extract the adjusted beginIndex from the VScroller result to update scroll position
			// We need to call SliceVertical to get the adjusted beginIndex
			sliceResult := layout.SliceVertical(allItemNodes, props.ScrollOffset, props.SelectedIndex, availableHeight)
			if props.OnUpdateScrollPos != nil && sliceResult.BeginIndex != props.ScrollOffset {
				props.OnUpdateScrollPos(sliceResult.BeginIndex)
			}

			// Combine header and scroller content
			return dom.Fragment(append(headerNodes, scrollerNode)...)
		}(),
	)
}

func renderMaterialItem(index int, material *models.LearningMaterial, isSelected bool) *dom.Node {
	// Format the title with index and selection indicator
	prefix := "  "
	if isSelected {
		prefix = "> "
	}
	titleText := fmt.Sprintf("%s%d. %s", prefix, index, material.Title)

	// Format metadata
	metaText := fmt.Sprintf("   Type: %s | Difficulty: %s | Source: %s",
		material.Type, material.Difficulty, material.Source)

	// Format description if available
	var descNode *dom.Node
	if material.Description != "" {
		descNode = dom.Div(dom.DivProps{},
			dom.Text("   "+material.Description, styles.Style{
				Color: colors.TextSecondary,
			}),
		)
	}

	// Format timestamps
	timeText := fmt.Sprintf("   Created: %s | Updated: %s",
		material.CreateTime.Format("2006-01-02 15:04"),
		material.UpdateTime.Format("2006-01-02 15:04"))

	titleStyle := styles.Style{
		Bold: true,
	}
	if isSelected {
		titleStyle.Color = "2" // Green for selected
		titleStyle.Bold = true
	}

	return dom.Div(dom.DivProps{},
		dom.Div(dom.DivProps{},
			dom.Text(titleText, titleStyle),
		),
		dom.Div(dom.DivProps{},
			dom.Text(metaText, styles.Style{
				Color: colors.TextMetadata,
			}),
		),
		func() *dom.Node {
			if descNode != nil {
				return descNode
			}
			return nil
		}(),
		dom.Div(dom.DivProps{},
			dom.Text(timeText, styles.Style{
				Color: colors.TextSecondary,
			}),
		),
		dom.Div(dom.DivProps{}, dom.Text("")), // Empty line for spacing between items
	)
}
