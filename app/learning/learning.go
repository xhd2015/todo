package learning

import (
	"fmt"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models"
)

type LearningMaterialListProps struct {
	Materials        []*models.LearningMaterial
	SelectedIndex    int
	OnNavigateBack   func()
	OnReload         func()
	OnSelectMaterial func(index int)
	OnOpenMaterial   func(materialID int64)
	OnNavigateUp     func()
	OnNavigateDown   func()
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
				}
				event.PreventDefault()
			case dom.KeyTypeUp:
				if props.OnNavigateUp != nil {
					props.OnNavigateUp()
				}
				event.PreventDefault()
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
		dom.Div(dom.DivProps{},
			dom.Text("Learning Materials (Last 10)", styles.Style{
				Bold: true,
			}),
		),
		dom.Div(dom.DivProps{},
			dom.Text("Press ↑/↓ to navigate, Enter to read, 'r' to reload, ESC to go back", styles.Style{
				Color: "8",
			}),
		),
		dom.Div(dom.DivProps{}, dom.Text("")), // Empty line for spacing
		func() *dom.Node {
			if len(props.Materials) == 0 {
				return dom.Div(dom.DivProps{},
					dom.Text("No learning materials found", styles.Style{
						Color: "8",
					}),
				)
			}

			children := make([]*dom.Node, 0, len(props.Materials))
			for i, material := range props.Materials {
				isSelected := i == props.SelectedIndex
				children = append(children, renderMaterialItem(i+1, material, isSelected))
			}
			return dom.Div(dom.DivProps{}, children...)
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
				Color: "8",
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
				Color: "6",
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
				Color: "8",
			}),
		),
		dom.Div(dom.DivProps{}, dom.Text("")), // Empty line for spacing between items
	)
}
