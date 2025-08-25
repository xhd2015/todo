package models

import (
	"time"

	"github.com/xhd2015/go-dom-tui/dom"
)

type InputState struct {
	Value          string
	Focused        bool
	CursorPosition int
	LastInputEvent *dom.DOMEvent
	LastInputTime  time.Time
}

func (c *InputState) Reset() {
	c.Value = ""
	c.CursorPosition = 0
}

func (c *InputState) FocusWithText(text string) {
	c.Focused = true
	c.Value = text
	c.CursorPosition = len([]rune(text))
}

type LogEntryView struct {
	Data *LogEntry

	MatchTexts []MatchText

	DetailPage *EntryOnDetailPage

	Notes    []*NoteView
	Children LogEntryViews

	// IncludeHistory controls whether history children are included
	// When true, shows (*) indicator and displays all children including history
	// toggled by 'v' command
	IncludeHistory bool

	// IncludeNotes controls whether notes are shown for this entry and its subtree
	// When true, shows notes for this entry and all its descendants
	// toggled by 'n' command
	IncludeNotes bool
}

type MatchText struct {
	Text  string
	Match bool
}

type LogEntryViews []*LogEntryView

type NoteView struct {
	Data       *Note
	MatchTexts []MatchText
}

type SelectedNoteMode int

const (
	SelectedNoteMode_Default SelectedNoteMode = iota
	SelectedNoteMode_Editing
	SelectedNoteMode_Deleting
)

type EntryOnDetailPage struct {
	SelectedNoteID int64

	SelectedNoteMode SelectedNoteMode

	InputState InputState

	EditInputState InputState

	ConfirmDeleteButton int

	SelectedChildEntryID int64
}

func (list LogEntryViews) Get(id int64) *LogEntryView {
	for _, e := range list {
		if e.Data.ID == id {
			return e
		}
		found := e.Children.Get(id)
		if found != nil {
			return found
		}
	}
	return nil
}

func (list LogEntryViews) FindNextOrLast(id int64) *LogEntryView {
	found := list.findAdjacent(id, true)
	if found != nil {
		return found
	}
	if len(list) > 0 {
		return list[len(list)-1]
	}
	return nil
}

func (list LogEntryViews) FindPrevOrFirst(id int64) *LogEntryView {
	found := list.findAdjacent(id, false)
	if found != nil {
		return found
	}
	if len(list) > 0 {
		return list[0]
	}
	return nil
}

func (list LogEntryViews) FindNext(id int64) *LogEntryView {
	return list.findAdjacent(id, true)
}

func (list LogEntryViews) FindPrev(id int64) *LogEntryView {
	return list.findAdjacent(id, false)
}

func (list LogEntryViews) findAdjacent(id int64, next bool) *LogEntryView {
	var traverse func(prev *LogEntryView, e *LogEntryView) *LogEntryView
	traverse = func(prev *LogEntryView, cur *LogEntryView) *LogEntryView {
		p := prev
		for _, child := range cur.Children {
			found := traverse(p, child)
			if found != nil {
				return found
			}
			p = child
		}
		if next {
			if prev != nil && prev.Data.ID == id {
				return cur
			}
		} else {
			if cur != nil && cur.Data.ID == id {
				return prev
			}
		}
		return nil
	}
	var prevID *LogEntryView
	for _, e := range list {
		found := traverse(prevID, e)
		if found != nil {
			return found
		}
		prevID = e
	}
	return nil
}
