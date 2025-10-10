package app

import (
	"github.com/xhd2015/go-dom-tui/dom"
)

const HEADER_HEIGHT = 9

func MainPage(state *State, window *dom.Window) *dom.Node {
	const RESERVE_ENTRY = 2
	const SPACE_BETWEEN_LIST_AND_INPUT = 2
	const INPUT_HEIGHT = 2
	const LINES_UNDER_INPUT = 3

	height := window.Height

	extraLines := getLines(state.SelectedEntryMode)

	maxEntries := height - HEADER_HEIGHT - RESERVE_ENTRY - SPACE_BETWEEN_LIST_AND_INPUT - INPUT_HEIGHT - LINES_UNDER_INPUT - extraLines
	// Minimum of 5 entries to ensure usability
	if maxEntries < 5 {
		maxEntries = 5
	}

	computeResult := computeVisibleEntries(state.Entries, maxEntries, state.SliceStart, state.SelectedEntryID, state.SelectFromSource, state.ZenMode, state.IsSearchActive, state.SearchQuery, state.ShowNotes)

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
		OnNavigate: func(e *dom.DOMEvent, entryType TreeEntryType, entryID int64, direction int) {
			// find index of current selected item (entry or note)
			index := -1

			if entryType == TreeEntryType_Note {
				// Looking for a note
				for i, wrapperEntry := range computeResult.FullEntries {
					if wrapperEntry.Type == TreeEntryType_Note && wrapperEntry.Note != nil && wrapperEntry.Note.Note.Data.ID == entryID {
						index = i
						break
					}
				}
			} else {
				// Looking for a log entry
				for i, wrapperEntry := range computeResult.FullEntries {
					if wrapperEntry.Type == TreeEntryType_Log && wrapperEntry.Log != nil && wrapperEntry.Log.Entry.Data.ID == entryID {
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
				if (nextEntry.Type == TreeEntryType_Log && nextEntry.Log != nil) ||
					(nextEntry.Type == TreeEntryType_Note && nextEntry.Note != nil) {
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
			if nextItem.Type == TreeEntryType_Log && nextItem.Log != nil {
				state.Select(nextItem.Log.Entry.Data.ID)
			} else if nextItem.Type == TreeEntryType_Note && nextItem.Note != nil {
				state.SelectNote(nextItem.Note.Note.Data.ID, nextItem.Note.EntryID)
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
				if wrapperEntry.Type == TreeEntryType_Log && wrapperEntry.Log != nil {
					state.Select(wrapperEntry.Log.Entry.Data.ID)
					break
				} else if wrapperEntry.Type == TreeEntryType_Note && wrapperEntry.Note != nil {
					state.SelectNote(wrapperEntry.Note.Note.Data.ID, wrapperEntry.Note.EntryID)
					break
				}
			}
		},
		OnGoToLast: func(e *dom.DOMEvent) {
			state.SliceStart = len(computeResult.FullEntries) - maxEntries
			// Find last selectable item (log entry or note)
			for i := len(computeResult.FullEntries) - 1; i >= 0; i-- {
				wrapperEntry := computeResult.FullEntries[i]
				if wrapperEntry.Type == TreeEntryType_Log && wrapperEntry.Log != nil {
					state.Select(wrapperEntry.Log.Entry.Data.ID)
					break
				} else if wrapperEntry.Type == TreeEntryType_Note && wrapperEntry.Note != nil {
					state.SelectNote(wrapperEntry.Note.Note.Data.ID, wrapperEntry.Note.EntryID)
					break
				}
			}
		},
		OnGoToTop: func(e *dom.DOMEvent) {
			// Find first selectable item in visible entries (log entry or note)
			for _, wrapperEntry := range computeResult.VisibleEntries {
				if wrapperEntry.Type == TreeEntryType_Log && wrapperEntry.Log != nil {
					state.Select(wrapperEntry.Log.Entry.Data.ID)
					break
				} else if wrapperEntry.Type == TreeEntryType_Note && wrapperEntry.Note != nil {
					state.SelectNote(wrapperEntry.Note.Note.Data.ID, wrapperEntry.Note.EntryID)
					break
				}
			}
		},
		OnGoToBottom: func(e *dom.DOMEvent) {
			// Find last selectable item in visible entries (log entry or note)
			for i := len(computeResult.VisibleEntries) - 1; i >= 0; i-- {
				wrapperEntry := computeResult.VisibleEntries[i]
				if wrapperEntry.Type == TreeEntryType_Log && wrapperEntry.Log != nil {
					state.Select(wrapperEntry.Log.Entry.Data.ID)
					break
				} else if wrapperEntry.Type == TreeEntryType_Note && wrapperEntry.Note != nil {
					state.SelectNote(wrapperEntry.Note.Note.Data.ID, wrapperEntry.Note.EntryID)
					break
				}
			}
		},
		OnEnter: func(e *dom.DOMEvent, entryType TreeEntryType, entryID int64) {
			// Navigate to the detail page of the entry that owns this note
			// entryType indicates whether this came from a log entry or note
			state.Routes.Push(DetailRoute(entryID))

			// Reset the input state for the target entry (same as regular todo items)
			targetEntry := state.Entries.Get(entryID)
			if targetEntry != nil {
				targetEntry.DetailPage.InputState.Value = ""
				targetEntry.DetailPage.InputState.Focused = true
				targetEntry.DetailPage.InputState.CursorPosition = 0
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
