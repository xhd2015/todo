package app

import (
	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
)

type MenuItem struct {
	Text     string
	Color    string
	OnSelect func()
}

type MenuProps struct {
	Title         string
	SelectedIndex int
	Items         []MenuItem
	OnSelect      func(index int)
	OnKeyDown     func(e *dom.DOMEvent)

	OnDismiss func()
}

func Menu(props MenuProps) *dom.Node {
	children := make([]*dom.Node, 0, len(props.Items))
	for i, item := range props.Items {
		selected := i == props.SelectedIndex
		style := styles.Style{
			BorderRouned: true,
			BorderColor:  colors.PURPLE_PRIMARY,
			Bold:         true,
			Color:        item.Color,
			FontSize:     2,
		}
		if selected {
			style.BackgroundColor = colors.PURPLE_PRIMARY
		}
		children = append(children, dom.Div(dom.DivProps{
			Style:     style,
			Focused:   selected,
			Focusable: true,
			OnKeyDown: func(e *dom.DOMEvent) {
				if e.Key == "up" {
					next := i - 1
					if next < 0 {
						next = len(props.Items) - 1
					}
					e.PreventDefault()
					props.OnSelect(next)
				}
				if e.Key == "down" {
					next := i + 1
					if next >= len(props.Items) {
						next = 0
					}
					e.PreventDefault()
					props.OnSelect(next)
				}
				if e.Key == "enter" {
					if item.OnSelect != nil {
						item.OnSelect()
					}
					if props.OnSelect != nil {
						props.OnSelect(i)
					}
				}
			},
		}, dom.Text(item.Text, styles.Style{
			Bold: selected,
		})))
	}
	return dom.Div(dom.DivProps{
		Style: styles.Style{
			BorderRouned: true,
			BorderColor:  colors.PURPLE_PRIMARY,
			NoDefault:    true,
		},
		Focusable: true,
		OnKeyDown: func(d *dom.DOMEvent) {
			if d.Key == "esc" {
				d.PreventDefault()
				if props.OnDismiss != nil {
					props.OnDismiss()
				}
			} else if props.OnKeyDown != nil {
				props.OnKeyDown(d)
			}
		},
	},
		dom.Div(dom.DivProps{
			Style: styles.Style{
				Color:    colors.GREY_TEXT,
				Italic:   true,
				FontSize: 1,
			},
		}, dom.Text(props.Title, styles.Style{
			Bold: true,
		})),
		dom.Fragment(
			children...,
		),
	)
}
