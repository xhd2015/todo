package app

import (
	"strings"

	"github.com/xhd2015/go-dom-tui/dom"
)

func MainPage(state *State, window *dom.Window) *dom.Node {
	const HEADER_HEIGHT = 3
	const INPUT_HEIGHT = 2
	const LINES_UNDER_INPUT = 3
	const RESERVE_ENTRY = 2
	const SPACE_BETWEEN_LIST_AND_INPUT = 2

	height := window.Height

	maxEntries := height - HEADER_HEIGHT - RESERVE_ENTRY - SPACE_BETWEEN_LIST_AND_INPUT - INPUT_HEIGHT - LINES_UNDER_INPUT
	// Minimum of 5 entries to ensure usability
	if maxEntries < 5 {
		maxEntries = 5
	}

	computeResult := computeVisibleEntries(state.Entries, maxEntries, state.SliceStart, state.ZenMode, state.IsSearchActive, state.SearchQuery)

	itemsHeight := len(computeResult.VisibleEntries)
	if computeResult.EntriesAbove > 0 {
		itemsHeight += 1
	}
	if computeResult.EntriesBelow > 0 {
		itemsHeight += 1
	}

	// Render the tree of entries
	children := RenderEntryTree(RenderEntryTreeProps{
		State:        state,
		EntriesAbove: computeResult.EntriesAbove,
		EntriesBelow: computeResult.EntriesBelow,
		Entries:      computeResult.VisibleEntries,
		OnNavigate: func(e *dom.DOMEvent, entryID int64, direction int) {
			// find index
			index := -1
			for i, entry := range computeResult.FullEntries {
				if entry.Entry.Data.ID == entryID {
					index = i
					break
				}
			}
			if index == -1 {
				return
			}

			next := index + direction
			if next < 0 || next >= len(computeResult.FullEntries) {
				if e.KeydownEvent != nil {
					if e.KeydownEvent.KeyType == dom.KeyTypeUp || e.KeydownEvent.KeyType == dom.KeyTypeDown {
						// fallback to default behavior
						return
					}
				}
				// loop around
				if next < 0 {
					next = len(computeResult.FullEntries) - 1
				} else if next >= len(computeResult.FullEntries) {
					next = 0
				}
			}
			e.PreventDefault()

			sliceStart := computeResult.EffectiveSliceStart
			sliceEnd := sliceStart + maxEntries

			state.SelectedEntryID = computeResult.FullEntries[next].Entry.Data.ID
			if next < sliceStart {
				state.SliceStart = next
			} else if next >= sliceEnd {
				state.SliceStart = next - maxEntries + 1
			}
		},
	})

	spaceHeight := height - HEADER_HEIGHT - itemsHeight - INPUT_HEIGHT - LINES_UNDER_INPUT
	var brs []*dom.Node
	if spaceHeight > 0 {
		brs = make([]*dom.Node, spaceHeight)
		for i := range brs {
			brs[i] = dom.Br()
		}
	}

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
