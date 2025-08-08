package app

import (
	"strings"

	"github.com/xhd2015/go-dom-tui/dom"
)

func MainInput(state *State, fullEntries []EntryWithDepth) *dom.Node {
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
					for _, entry := range fullEntries {
						if len(entry.Entry.MatchTexts) > 0 {
							foundID = entry.Entry.Data.ID
							break
						}
					}
					if foundID == 0 {
						foundID = fullEntries[0].Entry.Data.ID
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
				switch s {
				case "/history":
					// Toggle ShowHistory and refresh entries
					state.ShowHistory = !state.ShowHistory
					if state.OnRefreshEntries != nil {
						state.OnRefreshEntries()
					}
					return true
				case "/reload":
					if state.OnRefreshEntries != nil {
						state.OnRefreshEntries()
					}
					return true
				case "/zen":
					state.ZenMode = !state.ZenMode
					return true
				default:
					// Unknown command, do nothing
					return true
				}
			}

			state.OnAdd(s)
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
