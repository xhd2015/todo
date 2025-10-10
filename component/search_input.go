package component

import (
	"strings"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/todo/models"
)

type InputProps struct {
	Placeholder        string
	State              *models.InputState
	OnEnter            func(string) bool
	OnSearchChange     func(string)
	OnSearchActivate   func()
	OnSearchDeactivate func()
	OnKeyDown          func(event *dom.DOMEvent) bool
	InputType          string
	Width              int // Allow customizable width
}

func SearchInput(props InputProps) *dom.Node {
	// Default width if not specified
	width := props.Width
	if width == 0 {
		width = 50 // Default width
	}

	return dom.Input(dom.InputProps{
		Placeholder:    props.Placeholder,
		Value:          props.State.Value,
		Focused:        props.State.Focused,
		CursorPosition: props.State.CursorPosition,
		Focusable:      dom.Focusable(true),
		Width:          width,
		OnFocus: func() {
			props.State.Focused = true
		},
		OnBlur: func() {
			props.State.Focused = false
		},
		InputType: props.InputType,
		OnChange: func(value string) {
			props.State.Value = value

			// Handle search functionality if callbacks are provided
			if props.OnSearchActivate != nil && props.OnSearchChange != nil && props.OnSearchDeactivate != nil {
				if strings.HasPrefix(value, "?") {
					props.OnSearchActivate()
					// Extract search query (remove the ? prefix)
					query := strings.TrimPrefix(value, "?")
					props.OnSearchChange(query)
				} else {
					// Not a search query, deactivate search if active
					props.OnSearchDeactivate()
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
				if props.OnKeyDown(event) {
					return
				}
			}
			keyEvent := event.KeydownEvent
			switch keyEvent.KeyType {
			case dom.KeyTypeEnter:
				if props.State.Value == "" {
					return
				}
				if props.OnEnter != nil && props.OnEnter(props.State.Value) {
					props.State.Value = ""
					props.State.CursorPosition = 0
				}
			case dom.KeyTypeEsc:
				// Exit search mode if active and search callbacks are provided
				if props.OnSearchDeactivate != nil && strings.HasPrefix(props.State.Value, "?") {
					props.OnSearchDeactivate()
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
