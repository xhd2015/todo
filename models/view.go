package models

import "time"

type InputState struct {
	Value          string
	Focused        bool
	CursorPosition int
}

type EntryView struct {
	ID   int64
	Text string

	Done bool

	CreateTime time.Time
	UpdateTime time.Time

	DetailPage *EntryOnDetailPage

	Notes []*NoteView
}

type NoteView struct {
	ID   int64
	Text string
}

type EntryOnDetailPage struct {
	InputState *InputState
}
