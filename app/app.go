package app

import (
	"time"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"

	"github.com/xhd2015/todo/models/states"
)

const (
	CtrlCExitDelayMs = 1000
	UIWidth          = 50 // Shared width for status bar and input components
)

type State = states.State

type IDs []int64

func (ids *IDs) Pop() {
	*ids = (*ids)[:len(*ids)-1]
}

func (ids *IDs) Push(id int64) {
	*ids = append(*ids, id)
}

func (ids IDs) SetLast(id int64) {
	ids[len(ids)-1] = id
}

func (ids IDs) Last() int64 {
	return ids[len(ids)-1]
}

func App(state *State, window *dom.Window) *dom.Node {
	return dom.Div(dom.DivProps{
		OnKeyDown: func(event *dom.DOMEvent) {
			keyEvent := event.KeydownEvent
			if keyEvent == nil {
				return
			}
			switch keyEvent.KeyType {
			case dom.KeyTypeCtrlC:
				if time.Since(state.LastCtrlC) < time.Millisecond*CtrlCExitDelayMs {
					state.Quit()
					return
				}
				state.LastCtrlC = time.Now()

				go func() {
					time.Sleep(time.Millisecond * CtrlCExitDelayMs)
					state.Refresh()
				}()
			case dom.KeyTypeEsc:
				if len(state.Routes) > 0 {
					state.Routes.Pop()
				}
			}
		},
	},
		func() *dom.Node {
			title := "TODO List"
			if len(state.Routes) > 0 {
				last := state.Routes.Last()
				switch last.Type {
				case states.RouteType_Main:
					title = "Main"
				case states.RouteType_Detail:
					title = "Detail"
				case states.RouteType_Config:
					title = "Config"
				case states.RouteType_HappeningList:
					title = "Happenings"
				case states.RouteType_HumanState:
					title = "Human States"
				case states.RouteType_Help:
					title = "Help"
				case states.RouteType_Learning:
					title = "Learning Materials"
				case states.RouteType_Reading:
					title = "Reading"
				}
			}
			return dom.H1(dom.DivProps{}, dom.Text(title, styles.Style{
				Bold:        true,
				BorderColor: "orange",
			}))
		}(),
		func() *dom.Node {
			if len(state.Routes) == 0 {
				return RenderRoute(state, states.Route{Type: states.RouteType_Main}, window)
			} else {
				return RenderRoute(state, state.Routes.Last(), window)
			}
		}(),

		dom.Spacer(),
		func() *dom.Node {
			if time.Since(state.LastCtrlC) < time.Millisecond*CtrlCExitDelayMs {
				return dom.Text("press Ctrl-C again to exit", styles.Style{
					Bold:  true,
					Color: "1",
				})
			}
			return dom.Text("type 'exit','quit' or 'q' to exit")
		}(),
		AppStatusBar(state),
	)
}
