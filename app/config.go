package app

import (
	"fmt"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
)

func ConfigPage(state *State) *dom.Node {
	configState := state.Routes.Last().ConfigPage
	storageTypes := []string{"local file", "local sqlite", "server"}

	var configItems []*dom.Node

	// Storage Type selector
	configItems = append(configItems, dom.Div(dom.DivProps{},
		dom.Text("Storage Type:", styles.Style{Bold: true}),
	))

	for i, storageType := range storageTypes {
		selected := i == int(configState.PickingStorageType)
		style := styles.Style{}
		if selected {
			style.Color = "2"
			style.Bold = true
		}
		configItems = append(configItems, dom.TextWithProps(fmt.Sprintf("  [%s] %s", func() string {
			if selected {
				return "x"
			}
			return " "
		}(), storageType), dom.TextNodeProps{
			Style:     style,
			Focused:   configState.ConfigPhase == ConfigPhase_PickingStorageType && selected,
			Focusable: configState.ConfigPhase == ConfigPhase_PickingStorageType,
			OnFocus: func() {
				configState.PickingStorageType = StorageType(i)
			},
			OnKeyDown: func(d *dom.DOMEvent) {
				if d.KeydownEvent.KeyType == dom.KeyTypeEnter {
					configState.SelectedStorageType = StorageType(i)
					configState.ConfigPhase = ConfigPhase_PickingStorageDetail
				}
			},
		}))
		configItems = append(configItems, dom.Br())
	}

	// Show server-specific options when server is selected
	if configState.SelectedStorageType == 2 { // server
		focusable := configState.ConfigPhase == ConfigPhase_PickingStorageDetail

		var confirmBorderColor string
		var cancelBorderColor string
		if configState.ConfirmButtonFocused {
			confirmBorderColor = colors.GREEN_SUCCESS
		}
		if configState.CancelButtonFocused {
			cancelBorderColor = colors.RED_ERROR
		}

		configItems = append(configItems,
			dom.Text("Server Address:", styles.Style{Bold: true}),
			dom.Br(),
			SearchInput(InputProps{
				Placeholder: "Server Address",
				State:       &configState.ServerAddr,
			}),
			dom.Br(),
			dom.Text("Server Auth Token:", styles.Style{Bold: true}),
			dom.Br(),
			SearchInput(InputProps{
				Placeholder: "Server Auth Token",
				State:       &configState.ServerAuthToken,
				InputType:   "password",
			}),
			dom.Br(),
			dom.Div(dom.DivProps{},
				dom.TextWithProps("Save", dom.TextNodeProps{
					Style: styles.Style{
						Bold:        configState.ConfirmButtonFocused,
						BorderColor: confirmBorderColor,
					},
					Focused:   configState.ConfirmButtonFocused,
					Focusable: focusable,
					OnBlur: func() {
						configState.ConfirmButtonFocused = false
					},
					OnFocus: func() {
						configState.ConfirmButtonFocused = true
					},
					OnKeyDown: func(d *dom.DOMEvent) {
						if d.KeydownEvent.KeyType == dom.KeyTypeEnter {
							// TODO: just saved
						}
					},
				}),
				dom.Text("    "),
				dom.TextWithProps("Cancel", dom.TextNodeProps{
					Style: styles.Style{
						Bold:        configState.CancelButtonFocused,
						BorderColor: cancelBorderColor,
					},
					Focused:   configState.CancelButtonFocused,
					Focusable: focusable,
					OnFocus: func() {
						configState.CancelButtonFocused = true
					},
					OnBlur: func() {
						configState.CancelButtonFocused = false
					},
					OnKeyDown: func(d *dom.DOMEvent) {
						if d.KeydownEvent.KeyType == dom.KeyTypeEnter {
							configState.ConfigPhase = ConfigPhase_PickingStorageType
						}
					},
				}),
			),
		)
	}

	return dom.Div(dom.DivProps{},
		dom.H1(dom.DivProps{}, dom.Text("Config", styles.Style{
			Bold:  true,
			Color: "1",
		})),
		dom.Div(dom.DivProps{}, configItems...),
	)
}
