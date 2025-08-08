package data

import (
	"fmt"
	"sort"
	"time"

	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

type LogManager struct {
	LogEntryService storage.LogEntryService
	LogNoteService  storage.LogNoteService

	Entries []*models.LogEntryView
}

func NewLogManager(logEntryService storage.LogEntryService, logNoteService storage.LogNoteService) *LogManager {
	return &LogManager{
		LogEntryService: logEntryService,
		LogNoteService:  logNoteService,
	}
}

func (m *LogManager) InitWithHistory(showHistory bool) error {
	entries, err := loadEntries(m.LogEntryService, m.LogNoteService, showHistory)
	if err != nil {
		return err
	}
	m.Entries = entries
	return nil
}

func loadEntries(svc storage.LogEntryService, noteSvc storage.LogNoteService, showHistory bool) ([]*models.LogEntryView, error) {
	entries, _, err := svc.List(storage.LogEntryListOptions{})
	if err != nil {
		return nil, err
	}

	var filteredEntries []models.LogEntry
	if !showHistory {
		// Filter out entries that are done and have done_time before today
		today := time.Now().Truncate(24 * time.Hour)
		for _, entry := range entries {
			if entry.Done && entry.DoneTime != nil && entry.DoneTime.Before(today) {
				// Skip entries that are done and have done_time before today
				continue
			}
			filteredEntries = append(filteredEntries, entry)
		}
	} else {
		// Show all entries including historical ones
		filteredEntries = entries
	}

	var entriesView []*models.LogEntryView
	// Create a map for quick lookup
	entryMap := make(map[int64]*models.LogEntryView)

	for _, entry := range filteredEntries {
		notes, _, err := noteSvc.List(entry.ID, storage.LogNoteListOptions{})
		if err != nil {
			return nil, err
		}
		notesView := make([]*models.NoteView, 0, len(notes))
		for _, note := range notes {
			notesView = append(notesView, &models.NoteView{
				Data: &note,
			})
		}
		entryView := &models.LogEntryView{
			Data:     &entry,
			Notes:    notesView,
			Children: []*models.LogEntryView{},
			DetailPage: &models.EntryOnDetailPage{
				InputState: models.InputState{
					Value: entry.Text,
				},
			},
		}
		entryMap[entry.ID] = entryView
		entriesView = append(entriesView, entryView)
	}

	// Build parent-child relationships
	for _, entryView := range entriesView {
		if entryView.Data.ParentID != 0 {
			if parent, exists := entryMap[entryView.Data.ParentID]; exists {
				parent.Children = append(parent.Children, entryView)
			}
		}
	}

	sortEntries(entriesView)
	return entriesView, nil
}

// Init initializes with default behavior (no history)
func (m *LogManager) Init() error {
	return m.InitWithHistory(false)
}

func sortEntries(entries []*models.LogEntryView) {
	sort.Slice(entries, func(i, j int) bool {
		return !isNewer(entries[i], entries[j])
	})
}

func isNewer(a *models.LogEntryView, b *models.LogEntryView) bool {
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

func (m *LogManager) Add(entry models.LogEntry) (int64, error) {
	if entry.CreateTime.IsZero() {
		entry.CreateTime = time.Now()
	}
	if entry.UpdateTime.IsZero() {
		entry.UpdateTime = time.Now()
	}
	id, err := m.LogEntryService.Add(entry)
	if err != nil {
		return 0, err
	}
	entry.ID = id
	entryView := &models.LogEntryView{
		Data:     &entry,
		Notes:    []*models.NoteView{},
		Children: []*models.LogEntryView{},
		DetailPage: &models.EntryOnDetailPage{
			InputState: models.InputState{
				Value: entry.Text,
			},
		},
	}

	// If this entry has a parent, add it to parent's children
	if entry.ParentID != 0 {
		for _, existingEntry := range m.Entries {
			if existingEntry.Data.ID == entry.ParentID {
				existingEntry.Children = append(existingEntry.Children, entryView)
				break
			}
		}
	}

	m.Entries = append(m.Entries, entryView)
	return id, nil
}

func (m *LogManager) Update(id int64, entry models.LogEntryOptional) error {
	if entry.UpdateTime == nil {
		t := time.Now()
		entry.UpdateTime = &t
	}

	var targetEntry *models.LogEntryView
	var oldParentID int64

	// Find the target entry and remember its old parent
	for _, e := range m.Entries {
		if e.Data.ID == id {
			targetEntry = e
			oldParentID = e.Data.ParentID
			break
		}
	}

	if targetEntry == nil {
		return fmt.Errorf("entry with id %d not found", id)
	}

	err := m.LogEntryService.Update(id, entry)
	if err != nil {
		return err
	}

	var hasAdjustedTopTime bool
	var parentChanged bool

	// Update the entry data
	targetEntry.Data.Update(&entry)
	hasAdjustedTopTime = entry.AdjustedTopTime != nil

	// Handle parent-child relationship changes
	if entry.ParentID != nil {
		newParentID := *entry.ParentID
		if newParentID != oldParentID {
			parentChanged = true

			// Remove from old parent's children
			if oldParentID != 0 {
				for _, e := range m.Entries {
					if e.Data.ID == oldParentID {
						for i, child := range e.Children {
							if child.Data.ID == id {
								e.Children = append(e.Children[:i], e.Children[i+1:]...)
								break
							}
						}
						break
					}
				}
			}

			// Add to new parent's children
			if newParentID != 0 {
				for _, e := range m.Entries {
					if e.Data.ID == newParentID {
						e.Children = append(e.Children, targetEntry)
						break
					}
				}
			}
		}
	}

	if hasAdjustedTopTime || parentChanged {
		sortEntries(m.Entries)
	}
	return nil
}

func (m *LogManager) Delete(id int64) error {
	err := m.LogEntryService.Delete(id)
	if err != nil {
		return err
	}

	// bread-first search
	var traverse func(entries []*models.LogEntryView) ([]*models.LogEntryView, bool)
	traverse = func(entries []*models.LogEntryView) ([]*models.LogEntryView, bool) {
		for i, e := range entries {
			if e.Data.ID == id {
				newEntries := make([]*models.LogEntryView, len(entries)-1)
				copy(newEntries, entries[:i])
				copy(newEntries[i:], entries[i+1:])
				return newEntries, true
			}
		}
		for _, e := range entries {
			children, ok := traverse(e.Children)
			if ok {
				e.Children = children
				return entries, true
			}
		}
		return entries, false
	}

	m.Entries, _ = traverse(m.Entries)
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

func (m *LogManager) UpdateNote(entryID int64, noteID int64, note models.NoteOptional) error {
	err := m.LogNoteService.Update(entryID, noteID, note)
	if err != nil {
		return err
	}
	for _, entry := range m.Entries {
		if entry.Data.ID == entryID {
			for _, n := range entry.Notes {
				if n.Data.ID == noteID {
					n.Data.Update(&note)
					return nil
				}
			}
		}
	}
	return nil
}

func (m *LogManager) Move(id int64, newParentID int64) error {
	err := m.LogEntryService.Move(id, newParentID)
	if err != nil {
		return err
	}

	// Update the in-memory representation
	for _, entry := range m.Entries {
		if entry.Data.ID == id {
			entry.Data.ParentID = newParentID
			entry.Data.UpdateTime = time.Now()
			break
		}
	}

	// Re-sort entries to ensure correct tree structure
	sortEntries(m.Entries)
	return nil
}
