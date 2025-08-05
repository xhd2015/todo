package app

import (
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/todo/models"
)

type InputProps struct {
	Placeholder string
	State       *models.InputState
	onEnter     func(string) bool
}

func BindInput(props InputProps) *dom.Node {
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
			if event.Key == "enter" {
				if props.State.Value == "" {
					return
				}
				if props.onEnter(props.State.Value) {
					props.State.Value = ""
					props.State.CursorPosition = 0
				}
			}
		},
	})
}
