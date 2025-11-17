package dialog

import (
	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
)

// ConfirmDialogProps contains the properties for the confirmation dialog
type ConfirmDialogProps struct {
	SelectedButton  int
	PromptText      string // e.g., "Delete todo?"
	DeleteText      string // e.g., "[Delete]" or "[OK]"
	CancelText      string // e.g., "[Cancel]"
	OnDelete        func()
	OnCancel        func()
	OnNavigateRight func()
	OnNavigateLeft  func()
}

// ConfirmDialog creates a confirmation dialog with action and cancel buttons
func ConfirmDialog(props ConfirmDialogProps) *dom.Node {
	// Set defaults
	promptText := props.PromptText
	if promptText == "" {
		promptText = "Delete todo?"
	}

	deleteText := props.DeleteText
	if deleteText == "" {
		deleteText = "[OK]"
	}

	cancelText := props.CancelText
	if cancelText == "" {
		cancelText = "[Cancel]"
	}

	return dom.Div(dom.DivProps{
		Style: styles.Style{},
	},
		dom.TextWithProps(promptText, dom.TextNodeProps{
			Style: styles.Style{},
		}),
		dom.TextWithProps(deleteText, dom.TextNodeProps{
			Focused:   props.SelectedButton == 0,
			Focusable: true,
			Style: styles.Style{
				Color: colors.RED_ERROR,
				Bold:  props.SelectedButton == 0,
			},
			OnKeyDown: func(d *dom.DOMEvent) {
				keyEvent := d.KeydownEvent
				switch keyEvent.KeyType {
				case dom.KeyTypeEsc:
					props.OnCancel()
				case dom.KeyTypeRight:
					props.OnNavigateRight()
				case dom.KeyTypeEnter:
					props.OnDelete()
				}
			},
		}),
		dom.TextWithProps(cancelText, dom.TextNodeProps{
			Focused:   props.SelectedButton == 1,
			Focusable: true,
			Style: styles.Style{
				Color: "blue",
				Bold:  props.SelectedButton == 1,
			},
			OnKeyDown: func(d *dom.DOMEvent) {
				keyEvent := d.KeydownEvent
				switch keyEvent.KeyType {
				case dom.KeyTypeEsc:
					props.OnCancel()
				case dom.KeyTypeLeft:
					props.OnNavigateLeft()
				case dom.KeyTypeEnter:
					props.OnCancel()
				}
			},
		}),
	)
}
