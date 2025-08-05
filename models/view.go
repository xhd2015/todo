package models

type InputState struct {
	Value          string
	Focused        bool
	CursorPosition int
}

type EntryView struct {
	Data *LogEntry

	DetailPage *EntryOnDetailPage

	Notes    []*NoteView
	Children []*EntryView
}

type NoteView struct {
	Data *Note
}

type EntryOnDetailPage struct {
	InputState *InputState
}
