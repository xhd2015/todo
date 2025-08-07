package app

import (
	"strings"

	"github.com/xhd2015/go-dom-tui/dom"
)

func MainPage(state *State, window *dom.Window) *dom.Node {
	height := window.Height
	availableHeight := height - 5 - len(state.Entries)
	if availableHeight < 3 {
		availableHeight = 3
	}
	var brs []*dom.Node
	if availableHeight > 3 {
		brs = make([]*dom.Node, availableHeight-3)
		for i := range brs {
			brs[i] = dom.Br()
		}
	}

	// Render the tree of entries
	children := RenderEntryTree(state)
	return dom.Fragment(
		dom.Ul(dom.DivProps{}, children...),
		dom.Fragment(brs...),
		// input
		func() *dom.Node {
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
							state.SelectedEntryID = state.LastSelectedEntryID
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
						// In search mode, enter just exits search
						state.IsSearchActive = false
						state.SearchQuery = ""
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
		}(),
	)
}
