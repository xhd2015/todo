package app

import (
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/todo/component"
)

// InputProps is an alias for component.InputProps for backward compatibility
type InputProps = component.InputProps

// SearchInput wraps the component.SearchInput with app-specific defaults
func SearchInput(props InputProps) *dom.Node {
	// Set the width to UIWidth if not specified
	if props.Width == 0 {
		props.Width = UIWidth
	}

	return component.SearchInput(props)
}
