package storage

import "github.com/xhd2015/todo/models"

type LogEntryListOptions struct {
	Filter    string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}

type LogEntryService interface {
	List(options LogEntryListOptions) ([]models.LogEntry, int64, error)
	Add(entry models.LogEntry) (int64, error)
	Delete(id int64) error
	Update(id int64, update models.LogEntryOptional) error
	Move(id int64, newParentID int64) error
}

type LogNoteListOptions struct {
	Filter    string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}

type LogNoteService interface {
	List(entryID int64, options LogNoteListOptions) ([]models.Note, int64, error)
	ListForEntries(entryIDs []int64) (map[int64][]models.Note, error)
	Add(entryID int64, note models.Note) (int64, error)
	Delete(entryID int64, noteID int64) error
	Update(entryID int64, noteID int64, update models.NoteOptional) error
}
