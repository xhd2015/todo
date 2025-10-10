package models

import (
	"time"
)

type LogEntry struct {
	ID              int64      `json:"id"`
	Text            string     `json:"text"`
	Done            bool       `json:"done"`
	DoneTime        *time.Time `json:"done_time"`
	CreateTime      time.Time  `json:"create_time"`
	UpdateTime      time.Time  `json:"update_time"`
	AdjustedTopTime int64      `json:"adjusted_top_time"`
	HighlightLevel  int        `json:"highlight_level"`
	ParentID        int64      `json:"parent_id"`
}

type LogEntryOptional struct {
	ID              *int64      `json:"id"`
	Text            *string     `json:"text"`
	Done            *bool       `json:"done"`
	DoneTime        **time.Time `json:"done_time"`
	CreateTime      *time.Time  `json:"create_time"`
	UpdateTime      *time.Time  `json:"update_time"`
	AdjustedTopTime *int64      `json:"adjusted_top_time"`
	HighlightLevel  *int        `json:"highlight_level"`
	ParentID        *int64      `json:"parent_id"`
}

func (c *LogEntry) Update(optional *LogEntryOptional) {
	if optional == nil {
		return
	}
	if optional.ID != nil {
		c.ID = *optional.ID
	}
	if optional.Text != nil {
		c.Text = *optional.Text
	}
	if optional.Done != nil {
		c.Done = *optional.Done
	}
	if optional.DoneTime != nil {
		c.DoneTime = *optional.DoneTime
	}
	if optional.CreateTime != nil {
		c.CreateTime = *optional.CreateTime
	}
	if optional.UpdateTime != nil {
		c.UpdateTime = *optional.UpdateTime
	}
	if optional.AdjustedTopTime != nil {
		c.AdjustedTopTime = *optional.AdjustedTopTime
	}
	if optional.HighlightLevel != nil {
		c.HighlightLevel = *optional.HighlightLevel
	}
	if optional.ParentID != nil {
		c.ParentID = *optional.ParentID
	}
}

type Config struct {
	LastInput  string `json:"last_input"`
	RunningPID int    `json:"running_pid"`
	// value: sqlite(default), file, server
	StorageType string `json:"storage_type,omitempty"`

	// server_addr and server_token are only used when storage_type is server
	ServerAddr  string `json:"server_addr,omitempty"`
	ServerToken string `json:"server_token,omitempty"`
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

func (c *Note) Update(optional *NoteOptional) {
	if optional == nil {
		return
	}
	if optional.ID != nil {
		c.ID = *optional.ID
	}
	if optional.EntryID != nil {
		c.EntryID = *optional.EntryID
	}
	if optional.Text != nil {
		c.Text = *optional.Text
	}
	if optional.CreateTime != nil {
		c.CreateTime = *optional.CreateTime
	}
	if optional.UpdateTime != nil {
		c.UpdateTime = *optional.UpdateTime
	}
}
