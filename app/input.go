package app

import (
	"strings"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/todo/models"
)

type InputProps struct {
	Placeholder        string
	State              *models.InputState
	OnEnter            func(string) bool
	onSearchChange     func(string)
	onSearchActivate   func()
	onSearchDeactivate func()
	OnKeyDown          func(event *dom.DOMEvent)
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
		OnCursorMove: func(position int) {
			if position < 0 {
				position = 0
			}
			rnLen := runLength(props.State.Value)
			if position > rnLen+1 {
				position = rnLen + 1
			}
			props.State.CursorPosition = position
		},
		OnKeyDown: func(event *dom.DOMEvent) {
			if props.OnKeyDown != nil {
				props.OnKeyDown(event)
				return
			}
			keyEvent := event.KeydownEvent
			switch keyEvent.KeyType {
			case dom.KeyTypeEnter:
				if props.State.Value == "" {
					return
				}
				if props.OnEnter(props.State.Value) {
					props.State.Value = ""
					props.State.CursorPosition = 0
				}
			case dom.KeyTypeEsc:
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

func runLength(s string) int {
	return len([]rune(s))
}
