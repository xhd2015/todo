package app

import (
	"fmt"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/data"
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/models/states"
)

// loadConfigPageState loads the current config from file and converts to ConfigPageState
func loadConfigPageState() states.ConfigPageState {
	savedConfig, err := data.LoadConfig()
	if err != nil || savedConfig == nil {
		// Return default state if no config or error
		return states.ConfigPageState{
			ConfigPhase:         states.ConfigPhase_PickingStorageType,
			SelectedStorageType: states.StorageType_LocalSqlite,
			PickingStorageType:  states.StorageType_LocalSqlite,
		}
	}

	// Convert storage type from config
	var storageType states.StorageType
	switch savedConfig.StorageType {
	case "file":
		storageType = states.StorageType_LocalFile
	case "server":
		storageType = states.StorageType_Server
	default: // "sqlite" or empty
		storageType = states.StorageType_LocalSqlite
	}

	return states.ConfigPageState{
		ConfigPhase:         states.ConfigPhase_PickingStorageType,
		SelectedStorageType: storageType,
		PickingStorageType:  storageType,
		ServerAddr: models.InputState{
			Value: savedConfig.ServerAddr,
		},
		ServerAuthToken: models.InputState{
			Value: savedConfig.ServerToken,
		},
	}
}

// saveConfigPageState saves the current ConfigPageState to file
func saveConfigPageState(configState *states.ConfigPageState) error {
	// Load existing config to preserve other fields
	savedConfig, err := data.LoadConfig()
	if err != nil {
		return err
	}
	if savedConfig == nil {
		savedConfig = &models.Config{}
	}

	// Convert storage type to string
	switch configState.SelectedStorageType {
	case states.StorageType_LocalFile:
		savedConfig.StorageType = "file"
	case states.StorageType_Server:
		savedConfig.StorageType = "server"
	default: // StorageType_LocalSqlite
		savedConfig.StorageType = "sqlite"
	}

	// Update server settings
	savedConfig.ServerAddr = configState.ServerAddr.Value
	savedConfig.ServerToken = configState.ServerAuthToken.Value

	// Save back to file
	return data.SaveConfig(savedConfig)
}

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
			Focused:   configState.ConfigPhase == states.ConfigPhase_PickingStorageType && selected,
			Focusable: configState.ConfigPhase == states.ConfigPhase_PickingStorageType,
			OnFocus: func() {
				configState.PickingStorageType = states.StorageType(i)
			},
			OnKeyDown: func(d *dom.DOMEvent) {
				if d.KeydownEvent.KeyType == dom.KeyTypeEnter {
					configState.SelectedStorageType = states.StorageType(i)
					configState.ConfigPhase = states.ConfigPhase_PickingStorageDetail
				}
			},
		}))
		configItems = append(configItems, dom.Br())
	}

	// Show server-specific options when server is selected
	if configState.SelectedStorageType == 2 { // server
		focusable := configState.ConfigPhase == states.ConfigPhase_PickingStorageDetail

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
						switch d.KeydownEvent.KeyType {
						case dom.KeyTypeEnter:
							// Save config to file
							err := saveConfigPageState(configState)
							if err != nil {
								// TODO: Handle error properly - maybe show in status bar
								fmt.Printf("Error saving config: %v\n", err)
							}
							// Go back to main page
							state.Routes.Pop()
						case dom.KeyTypeRight:
							// Move focus to Cancel button
							configState.ConfirmButtonFocused = false
							configState.CancelButtonFocused = true
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
						switch d.KeydownEvent.KeyType {
						case dom.KeyTypeEnter:
							// Cancel - go back to main page without saving
							state.Routes.Pop()
						case dom.KeyTypeLeft:
							// Move focus to Save button
							configState.CancelButtonFocused = false
							configState.ConfirmButtonFocused = true
						}
					},
				}),
			),
		)
	}

	return dom.Div(dom.DivProps{},
		dom.Div(dom.DivProps{}, configItems...),
	)
}
