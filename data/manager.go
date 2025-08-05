package data

import (
	"sort"
	"time"

	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

type LogManager struct {
	LogEntryService storage.LogEntryService
	LogNoteService  storage.LogNoteService

	Entries []*models.EntryView
}

func NewLogManager(logEntryService storage.LogEntryService, logNoteService storage.LogNoteService) *LogManager {
	return &LogManager{
		LogEntryService: logEntryService,
		LogNoteService:  logNoteService,
	}
}

func (m *LogManager) Init() error {
	entries, _, err := m.LogEntryService.List(storage.LogEntryListOptions{})
	if err != nil {
		return err
	}
	for _, entry := range entries {
		notes, _, err := m.LogNoteService.List(entry.ID, storage.LogNoteListOptions{})
		if err != nil {
			return err
		}
		notesView := make([]*models.NoteView, 0, len(notes))
		for _, note := range notes {
			notesView = append(notesView, &models.NoteView{
				Data: &note,
			})
		}
		m.Entries = append(m.Entries, &models.EntryView{
			Data:  &entry,
			Notes: notesView,
			DetailPage: &models.EntryOnDetailPage{
				InputState: &models.InputState{
					Value: entry.Text,
				},
			},
		})
	}
	sortEntries(m.Entries)
	return nil
}

func sortEntries(entries []*models.EntryView) {
	sort.Slice(entries, func(i, j int) bool {
		return !isNewer(entries[i], entries[j])
	})
}

func isNewer(a *models.EntryView, b *models.EntryView) bool {
	// compare create time if both are not adjusted top time
	if a.Data.AdjustedTopTime == 0 && b.Data.AdjustedTopTime == 0 {
		return a.Data.CreateTime.After(b.Data.CreateTime)
	}

	// compare adjusted top time if both are adjusted top time
	if a.Data.AdjustedTopTime == 0 {
		return false
	}
	if b.Data.AdjustedTopTime == 0 {
		return true
	}

	// compare adjusted top time if both are adjusted top time
	return a.Data.AdjustedTopTime > b.Data.AdjustedTopTime
}

func (m *LogManager) Add(entry models.LogEntry) error {
	if entry.CreateTime.IsZero() {
		entry.CreateTime = time.Now()
	}
	if entry.UpdateTime.IsZero() {
		entry.UpdateTime = time.Now()
	}
	id, err := m.LogEntryService.Add(entry)
	if err != nil {
		return err
	}
	entry.ID = id
	m.Entries = append(m.Entries, &models.EntryView{
		Data:  &entry,
		Notes: []*models.NoteView{},
		DetailPage: &models.EntryOnDetailPage{
			InputState: &models.InputState{
				Value: entry.Text,
			},
		},
	})
	return nil
}

func (m *LogManager) Update(id int64, entry models.LogEntryOptional) error {
	if entry.UpdateTime == nil {
		t := time.Now()
		entry.UpdateTime = &t
	}
	err := m.LogEntryService.Update(id, entry)
	if err != nil {
		return err
	}
	var hasAdjustedTopTime bool
	for _, e := range m.Entries {
		if e.Data.ID == id {
			e.Data.Update(&entry)
			hasAdjustedTopTime = entry.AdjustedTopTime != nil
		}
	}
	if hasAdjustedTopTime {
		sortEntries(m.Entries)
	}
	return nil
}

func (m *LogManager) Delete(id int64) error {
	err := m.LogEntryService.Delete(id)
	if err != nil {
		return err
	}
	for i, e := range m.Entries {
		if e.Data.ID == id {
			m.Entries = append(m.Entries[:i], m.Entries[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *LogManager) AddNote(entryID int64, note models.Note) error {
	if note.CreateTime.IsZero() {
		note.CreateTime = time.Now()
	}
	if note.UpdateTime.IsZero() {
		note.UpdateTime = time.Now()
	}
	id, err := m.LogNoteService.Add(entryID, note)
	if err != nil {
		return err
	}
	note.ID = id
	for _, entry := range m.Entries {
		if entry.Data.ID == entryID {
			entry.Notes = append(entry.Notes, &models.NoteView{
				Data: &note,
			})
			return nil
		}
	}
	return nil
}

func (m *LogManager) DeleteNote(entryID int64, noteID int64) error {
	err := m.LogNoteService.Delete(entryID, noteID)
	if err != nil {
		return err
	}
	for _, entry := range m.Entries {
		if entry.Data.ID == entryID {
			for i, n := range entry.Notes {
				if n.Data.ID == noteID {
					entry.Notes = append(entry.Notes[:i], entry.Notes[i+1:]...)
					return nil
				}
			}
		}
	}
	return nil
}

func (m *LogManager) UpdateNote(entryID int64, noteID int64, note models.Note) error {
	err := m.LogNoteService.Update(entryID, noteID, models.NoteOptional{
		ID:         &noteID,
		Text:       &note.Text,
		CreateTime: &note.CreateTime,
		UpdateTime: &note.UpdateTime,
	})
	if err != nil {
		return err
	}
	for _, entry := range m.Entries {
		if entry.Data.ID == entryID {
			for i, n := range entry.Notes {
				if n.Data.ID == noteID {
					entry.Notes[i] = &models.NoteView{
						Data: &note,
					}
					return nil
				}
			}
		}
	}
	return nil
}
