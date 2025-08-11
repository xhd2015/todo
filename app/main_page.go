package app

import (
	"github.com/xhd2015/go-dom-tui/dom"
)

func MainPage(state *State, window *dom.Window) *dom.Node {
	const HEADER_HEIGHT = 3
	const INPUT_HEIGHT = 2
	const LINES_UNDER_INPUT = 3
	const RESERVE_ENTRY = 2
	const SPACE_BETWEEN_LIST_AND_INPUT = 2

	height := window.Height

	extraLines := getLines(state.SelectedEntryMode)

	maxEntries := height - HEADER_HEIGHT - RESERVE_ENTRY - SPACE_BETWEEN_LIST_AND_INPUT - INPUT_HEIGHT - LINES_UNDER_INPUT - extraLines
	// Minimum of 5 entries to ensure usability
	if maxEntries < 5 {
		maxEntries = 5
	}

	computeResult := computeVisibleEntries(state.Entries, maxEntries, state.SliceStart, state.SelectedEntryID, state.SelectFromSource, state.ZenMode, state.IsSearchActive, state.SearchQuery)

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
			if state.SliceStart == -1 {
				state.SliceStart = sliceStart
			}
			sliceEnd := sliceStart + maxEntries
			if sliceEnd > len(computeResult.FullEntries) {
				sliceEnd = len(computeResult.FullEntries)
			}

			state.Select(computeResult.FullEntries[next].Entry.Data.ID)
			state.SelectFromSource = SelectedSource_NavigateByKey
			if next < sliceStart {
				state.SliceStart = next
			} else if next >= sliceEnd {
				state.SliceStart = next - maxEntries + 1
			}
		},
		OnGoToFirst: func(e *dom.DOMEvent) {
			state.SliceStart = 0
			state.Select(computeResult.FullEntries[0].Entry.Data.ID)
		},
		OnGoToLast: func(e *dom.DOMEvent) {
			state.SliceStart = len(computeResult.FullEntries) - maxEntries
			state.Select(computeResult.FullEntries[len(computeResult.FullEntries)-1].Entry.Data.ID)
		},
		OnGoToTop: func(e *dom.DOMEvent) {
			state.Select(computeResult.VisibleEntries[0].Entry.Data.ID)
		},
		OnGoToBottom: func(e *dom.DOMEvent) {
			state.Select(computeResult.VisibleEntries[len(computeResult.VisibleEntries)-1].Entry.Data.ID)
		},
	})

	spaceHeight := height - HEADER_HEIGHT - itemsHeight - INPUT_HEIGHT - LINES_UNDER_INPUT - extraLines
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
		MainInput(state, computeResult.FullEntries),
	)
}
