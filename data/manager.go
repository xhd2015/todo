package data

import (
	idata "github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

type LogManager struct {
	LogEntryService idata.LogEntryService
	LogNoteService  idata.LogNoteService

	Entries []*models.EntryView
}

func NewLogManager(logEntryService idata.LogEntryService, logNoteService idata.LogNoteService) *LogManager {
	return &LogManager{
		LogEntryService: logEntryService,
		LogNoteService:  logNoteService,
	}
}

func (m *LogManager) Init() error {
	entries, _, err := m.LogEntryService.List(idata.LogEntryListOptions{})
	if err != nil {
		return err
	}
	for _, entry := range entries {
		notes, _, err := m.LogNoteService.List(entry.ID, idata.LogNoteListOptions{})
		if err != nil {
			return err
		}
		notesView := make([]*models.NoteView, len(notes))
		for i, note := range notes {
			notesView[i] = &models.NoteView{
				ID:   note.ID,
				Text: note.Text,
			}
		}
		m.Entries = append(m.Entries, &models.EntryView{
			ID:    entry.ID,
			Text:  entry.Text,
			Notes: notesView,
			DetailPage: &models.EntryOnDetailPage{
				InputState: &models.InputState{
					Value: entry.Text,
				},
			},
		})
	}
	return nil
}

func (m *LogManager) Add(entry models.LogEntry) error {
	id, err := m.LogEntryService.Add(entry)
	if err != nil {
		return err
	}
	entry.ID = id
	m.Entries = append(m.Entries, &models.EntryView{
		ID:    entry.ID,
		Text:  entry.Text,
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
	err := m.LogEntryService.Update(id, entry)
	if err != nil {
		return err
	}
	for _, e := range m.Entries {
		if e.ID == id {
			if entry.Text != nil {
				e.Text = *entry.Text
			}
			if entry.Done != nil {
				e.Done = *entry.Done
			}
			if entry.CreateTime != nil {
				e.CreateTime = *entry.CreateTime
			}
			if entry.UpdateTime != nil {
				e.UpdateTime = *entry.UpdateTime
			}
		}
	}
	return nil
}

func (m *LogManager) Delete(id int64) error {
	err := m.LogEntryService.Delete(id)
	if err != nil {
		return err
	}
	for i, e := range m.Entries {
		if e.ID == id {
			m.Entries = append(m.Entries[:i], m.Entries[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *LogManager) AddNote(entryID int64, note models.Note) error {
	id, err := m.LogNoteService.Add(entryID, note)
	if err != nil {
		return err
	}
	note.ID = id
	for _, entry := range m.Entries {
		if entry.ID == entryID {
			entry.Notes = append(entry.Notes, &models.NoteView{
				ID:   id,
				Text: note.Text,
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
		if entry.ID == entryID {
			for i, n := range entry.Notes {
				if n.ID == noteID {
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
		if entry.ID == entryID {
			for i, n := range entry.Notes {
				if n.ID == noteID {
					entry.Notes[i] = &models.NoteView{
						ID:   note.ID,
						Text: note.Text,
					}
					return nil
				}
			}
		}
	}
	return nil
}
