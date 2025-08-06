package models

type InputState struct {
	Value          string
	Focused        bool
	CursorPosition int
}

type LogEntryView struct {
	Data *LogEntry

	DetailPage *EntryOnDetailPage

	Notes    []*NoteView
	Children LogEntryViews
}

type LogEntryViews []*LogEntryView

type NoteView struct {
	Data *Note
}

type EntryOnDetailPage struct {
	InputState *InputState
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
		if next {
			if prev.Data.ID == id {
				return cur
			}
		} else {
			if cur.Data.ID == id {
				return prev
			}
		}
		p := prev
		for _, child := range cur.Children {
			found := traverse(p, child)
			if found != nil {
				return found
			}
			p = child
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
