package models

import (
	"time"
)

type LogEntry struct {
	ID         int64     `json:"id"`
	Text       string    `json:"text"`
	Done       bool      `json:"done"`
	CreateTime time.Time `json:"create_time"`
	UpdateTime time.Time `json:"update_time"`
}

type LogEntryOptional struct {
	ID         *int64     `json:"id"`
	Text       *string    `json:"text"`
	Done       *bool      `json:"done"`
	CreateTime *time.Time `json:"create_time"`
	UpdateTime *time.Time `json:"update_time"`
}

type Config struct {
	LastInput  string `json:"last_input"`
	RunningPID int    `json:"running_pid"`
}

type LogEntryLegacy struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`
	TodoID    int       `json:"todo_id"`
	TodoData  LogEntry  `json:"todo_data"`
}

type Note struct {
	ID         int64     `json:"id"`
	EntryID    int64     `json:"entry_id"`
	Text       string    `json:"text"`
	CreateTime time.Time `json:"create_time"`
	UpdateTime time.Time `json:"update_time"`
}

type NoteOptional struct {
	ID         *int64     `json:"id"`
	EntryID    *int64     `json:"entry_id"`
	Text       *string    `json:"text"`
	CreateTime *time.Time `json:"create_time"`
	UpdateTime *time.Time `json:"update_time"`
}
