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
	children := Tree(TreeProps{
		State:        state,
		EntriesAbove: computeResult.EntriesAbove,
		EntriesBelow: computeResult.EntriesBelow,
		Entries:      computeResult.VisibleEntries,
		OnNavigate: func(e *dom.DOMEvent, entryID int64, direction int) {
			// find index of current selected item (entry or note)
			index := -1
			isNoteSelected := entryID < 0 // negative IDs indicate notes

			if isNoteSelected {
				// Looking for a note (entryID is negative note ID)
				noteID := -entryID
				for i, wrapperEntry := range computeResult.FullEntries {
					if wrapperEntry.Type == WrapperEntryType_Note && wrapperEntry.TreeNote != nil && wrapperEntry.TreeNote.Note.Data.ID == noteID {
						index = i
						break
					}
				}
			} else {
				// Looking for a log entry
				for i, wrapperEntry := range computeResult.FullEntries {
					if wrapperEntry.Type == WrapperEntryType_Log && wrapperEntry.TreeEntry != nil && wrapperEntry.TreeEntry.Entry.Data.ID == entryID {
						index = i
						break
					}
				}
			}

			if index == -1 {
				return
			}

			// Find next selectable item (both log entries and notes are selectable)
			next := index + direction
			for {
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

				// Check if this item is selectable (both log entries and notes are selectable)
				nextEntry := computeResult.FullEntries[next]
				if (nextEntry.Type == WrapperEntryType_Log && nextEntry.TreeEntry != nil) ||
					(nextEntry.Type == WrapperEntryType_Note && nextEntry.TreeNote != nil) {
					break
				}

				// Move to next item
				next += direction

				// Prevent infinite loop - if we've checked all items and none are selectable
				if next == index {
					return
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

			// Select the appropriate item
			nextItem := computeResult.FullEntries[next]
			if nextItem.Type == WrapperEntryType_Log && nextItem.TreeEntry != nil {
				state.Select(nextItem.TreeEntry.Entry.Data.ID)
			} else if nextItem.Type == WrapperEntryType_Note && nextItem.TreeNote != nil {
				state.SelectNote(nextItem.TreeNote.Note.Data.ID, nextItem.TreeNote.EntryID)
			}

			state.SelectFromSource = SelectedSource_NavigateByKey
			if next < sliceStart {
				state.SliceStart = next
			} else if next >= sliceEnd {
				state.SliceStart = next - maxEntries + 1
			}
		},
		OnGoToFirst: func(e *dom.DOMEvent) {
			state.SliceStart = 0
			// Find first selectable item (log entry or note)
			for _, wrapperEntry := range computeResult.FullEntries {
				if wrapperEntry.Type == WrapperEntryType_Log && wrapperEntry.TreeEntry != nil {
					state.Select(wrapperEntry.TreeEntry.Entry.Data.ID)
					break
				} else if wrapperEntry.Type == WrapperEntryType_Note && wrapperEntry.TreeNote != nil {
					state.SelectNote(wrapperEntry.TreeNote.Note.Data.ID, wrapperEntry.TreeNote.EntryID)
					break
				}
			}
		},
		OnGoToLast: func(e *dom.DOMEvent) {
			state.SliceStart = len(computeResult.FullEntries) - maxEntries
			// Find last selectable item (log entry or note)
			for i := len(computeResult.FullEntries) - 1; i >= 0; i-- {
				wrapperEntry := computeResult.FullEntries[i]
				if wrapperEntry.Type == WrapperEntryType_Log && wrapperEntry.TreeEntry != nil {
					state.Select(wrapperEntry.TreeEntry.Entry.Data.ID)
					break
				} else if wrapperEntry.Type == WrapperEntryType_Note && wrapperEntry.TreeNote != nil {
					state.SelectNote(wrapperEntry.TreeNote.Note.Data.ID, wrapperEntry.TreeNote.EntryID)
					break
				}
			}
		},
		OnGoToTop: func(e *dom.DOMEvent) {
			// Find first selectable item in visible entries (log entry or note)
			for _, wrapperEntry := range computeResult.VisibleEntries {
				if wrapperEntry.Type == WrapperEntryType_Log && wrapperEntry.TreeEntry != nil {
					state.Select(wrapperEntry.TreeEntry.Entry.Data.ID)
					break
				} else if wrapperEntry.Type == WrapperEntryType_Note && wrapperEntry.TreeNote != nil {
					state.SelectNote(wrapperEntry.TreeNote.Note.Data.ID, wrapperEntry.TreeNote.EntryID)
					break
				}
			}
		},
		OnGoToBottom: func(e *dom.DOMEvent) {
			// Find last selectable item in visible entries (log entry or note)
			for i := len(computeResult.VisibleEntries) - 1; i >= 0; i-- {
				wrapperEntry := computeResult.VisibleEntries[i]
				if wrapperEntry.Type == WrapperEntryType_Log && wrapperEntry.TreeEntry != nil {
					state.Select(wrapperEntry.TreeEntry.Entry.Data.ID)
					break
				} else if wrapperEntry.Type == WrapperEntryType_Note && wrapperEntry.TreeNote != nil {
					state.SelectNote(wrapperEntry.TreeNote.Note.Data.ID, wrapperEntry.TreeNote.EntryID)
					break
				}
			}
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
