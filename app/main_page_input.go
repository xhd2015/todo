package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/xhd2015/go-dom-tui/dom"
)

func MainInput(state *State, fullEntries []WrapperEntry) *dom.Node {
	placeholder := "add todo"
	if state.IsSearchActive {
		placeholder = "search todos (ESC to exit search)"
	}

	return SearchInput(InputProps{
		Placeholder: placeholder,
		State:       &state.Input,
		OnKeyDown: func(event *dom.DOMEvent) bool {
			keyEvent := event.KeydownEvent
			switch keyEvent.KeyType {
			case dom.KeyTypeUp:
				if state.LastSelectedEntryID != 0 {
					state.Select(state.LastSelectedEntryID)
					state.LastSelectedEntryID = 0
					state.Input.Focused = false
					event.PreventDefault()
				}
			case dom.KeyTypeEsc:
				if state.IsSearchActive {
					state.ClearSearch()
				}
			case dom.KeyTypeCtrlC:
				if state.IsSearchActive {
					state.ClearSearch()
					event.PreventDefault()
					event.StopPropagation()
				}
			}
			return false
		},
		OnEnter: func(s string) bool {
			state.StatusBar.Error = ""
			if strings.TrimSpace(s) == "" {
				return false
			}

			// Handle search mode
			if state.IsSearchActive {
				state.IsSearchActive = false
				state.SearchQuery = ""

				// if any entry is match, select the first one
				if len(fullEntries) > 0 {
					// first match
					var foundID int64
					for _, wrapperEntry := range fullEntries {
						if wrapperEntry.Type == WrapperEntryType_Log && wrapperEntry.TreeEntry != nil {
							if len(wrapperEntry.TreeEntry.Entry.MatchTexts) > 0 {
								foundID = wrapperEntry.TreeEntry.Entry.Data.ID
								break
							}
						}
					}
					if foundID == 0 {
						// Find first log entry
						for _, wrapperEntry := range fullEntries {
							if wrapperEntry.Type == WrapperEntryType_Log && wrapperEntry.TreeEntry != nil {
								foundID = wrapperEntry.TreeEntry.Entry.Data.ID
								break
							}
						}
					}
					state.Select(foundID)
					state.SelectFromSource = SelectedSource_Search
				}
				return true
			}

			if s == "exit" || s == "quit" || s == "q" {
				state.Quit()
				return true
			}

			// Handle special commands starting with /
			if strings.HasPrefix(s, "/") {
				// Handle /export command with filename
				if filename, found := strings.CutPrefix(s, "/export "); found {
					filename = strings.TrimSpace(filename)
					if filename == "" {
						state.StatusBar.Error = "export requires a filename: /export <filename.json>"
						return true
					}

					// Export visible entries
					err := ExportVisibleEntries(filename, fullEntries)
					if err != nil {
						state.StatusBar.Error = fmt.Sprintf("export failed: %v", err)
					} else {
						state.StatusBar.Error = fmt.Sprintf("exported %d entries to %s", len(fullEntries), filename)
					}
					return true
				}

				switch s {
				case "/history":
					// Toggle ShowHistory and refresh entries
					wasShowingHistory := state.ShowHistory
					state.ShowHistory = !state.ShowHistory

					// If we're turning off history mode, reset all v states
					if wasShowingHistory && !state.ShowHistory {
						state.ResetAllChildrenVisibility()
					}

					if state.RefreshEntries != nil {
						state.Enqueue(func(ctx context.Context) error {
							return state.RefreshEntries(ctx)
						})
					}
					return true
				case "/reload":
					if state.RefreshEntries != nil {
						state.Enqueue(func(ctx context.Context) error {
							return state.RefreshEntries(ctx)
						})
					}
					return true
				case "/zen":
					state.ZenMode = !state.ZenMode
					return true
				case "/config":
					// show config page
					configState := loadConfigPageState()
					state.Routes.Push(ConfigRoute(configState))
					return true
				default:
					// Unknown command, do nothing
					state.StatusBar.Error = "unknown command: " + s
					return true
				}
			}

			state.Enqueue(func(ctx context.Context) error {
				return state.OnAdd(s)
			})
			return true
		},
		onSearchChange: func(query string) {
			state.SearchQuery = query
		},
		onSearchActivate: func() {
			state.IsSearchActive = true
		},
		onSearchDeactivate: func() {
			state.IsSearchActive = false
			state.SearchQuery = ""
		},
	})
}
