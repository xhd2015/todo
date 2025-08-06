package app

import (
	"strings"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/todo/models"
)

type InputProps struct {
	Placeholder        string
	State              *models.InputState
	onEnter            func(string) bool
	onSearchChange     func(string)
	onSearchActivate   func()
	onSearchDeactivate func()
}

func SearchInput(props InputProps) *dom.Node {
	return dom.Input(dom.InputProps{
		Placeholder:    props.Placeholder,
		Value:          props.State.Value,
		Focused:        props.State.Focused,
		CursorPosition: props.State.CursorPosition,
		Focusable:      dom.Focusable(true),
		OnFocus: func() {
			props.State.Focused = true
		},
		OnBlur: func() {
			props.State.Focused = false
		},
		OnChange: func(value string) {
			props.State.Value = value

			// Handle search functionality if callbacks are provided
			if props.onSearchActivate != nil && props.onSearchChange != nil && props.onSearchDeactivate != nil {
				if strings.HasPrefix(value, "?") {
					props.onSearchActivate()
					// Extract search query (remove the ? prefix)
					query := strings.TrimPrefix(value, "?")
					props.onSearchChange(query)
				} else {
					// Not a search query, deactivate search if active
					props.onSearchDeactivate()
				}
			}
		},
		OnCursorMove: func(delta int, seek int) {
			newPosition := props.State.CursorPosition + delta
			if newPosition < 0 {
				newPosition = 0
			}
			if newPosition > len(props.State.Value)+1 {
				newPosition = len(props.State.Value) + 1
			}
			props.State.CursorPosition = newPosition
		},
		OnKeyDown: func(event *dom.DOMEvent) {
			switch event.Key {
			case "enter":
				if props.State.Value == "" {
					return
				}
				if props.onEnter(props.State.Value) {
					props.State.Value = ""
					props.State.CursorPosition = 0
				}
			case "esc":
				// Exit search mode if active and search callbacks are provided
				if props.onSearchDeactivate != nil && strings.HasPrefix(props.State.Value, "?") {
					props.onSearchDeactivate()
					props.State.Value = ""
					props.State.CursorPosition = 0
				}
			}
		},
	})
}
