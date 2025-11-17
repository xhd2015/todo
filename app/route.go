package app

import (
	"fmt"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models/states"
)

func RenderRoute(state *State, route states.Route, window *dom.Window) *dom.Node {
	// Fixed frame overhead: Title (1) + Help/Exit message (1) + Status bar (1) = 3 lines
	const FIXED_FRAME_HEIGHT = 3
	availableHeight := window.Height - FIXED_FRAME_HEIGHT
	if availableHeight < 5 {
		availableHeight = 5 // Minimum height
	}

	switch route.Type {
	case states.RouteType_Main:
		return MainPage(state, availableHeight)
	case states.RouteType_Detail:
		return DetailPage(state, route.DetailPage.EntryID)
	case states.RouteType_Config:
		return ConfigPage(state)
	case states.RouteType_HappeningList:
		return states.HappeningListPage(state, availableHeight)
	case states.RouteType_HumanState:
		return states.HumanStatePage(state)
	case states.RouteType_Help:
		return states.HelpPage(state, window)
	case states.RouteType_Learning:
		return states.LearningPage(state, window.Width, availableHeight)
	case states.RouteType_Reading:
		return states.ReadingPage(state, route.ReadingPage.MaterialID, window.Width, availableHeight)
	default:
		return dom.Text(fmt.Sprintf("unknown route: %d", route.Type), styles.Style{
			Bold:  true,
			Color: colors.RED_ERROR,
		})
	}
}
