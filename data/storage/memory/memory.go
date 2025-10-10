package memory

import (
	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

// MemoryDataStore implements DataStore interface for in-memory storage
type MemoryDataStore struct {
	logEntries map[int64]models.LogEntry
	notes      map[int64]models.Note
	happenings map[int64]models.Happening
	nextID     int64
}

// NewMemoryDataStore creates a new in-memory data store
func NewMemoryDataStore() *MemoryDataStore {
	return &MemoryDataStore{
		logEntries: make(map[int64]models.LogEntry),
		notes:      make(map[int64]models.Note),
		happenings: make(map[int64]models.Happening),
		nextID:     1,
	}
}

// Entry operations
func (mds *MemoryDataStore) GetAllEntries() []models.LogEntry {
	entries := make([]models.LogEntry, 0, len(mds.logEntries))
	for _, entry := range mds.logEntries {
		entries = append(entries, entry)
	}
	return entries
}

func (mds *MemoryDataStore) GetEntry(id int64) (models.LogEntry, bool) {
	entry, exists := mds.logEntries[id]
	return entry, exists
}

func (mds *MemoryDataStore) AddEntry(entry models.LogEntry) error {
	mds.logEntries[entry.ID] = entry
	return nil
}

func (mds *MemoryDataStore) UpdateEntry(id int64, entry models.LogEntry) error {
	mds.logEntries[id] = entry
	return nil
}

func (mds *MemoryDataStore) DeleteEntry(id int64) error {
	delete(mds.logEntries, id)
	return nil
}

// Note operations
func (mds *MemoryDataStore) GetAllNotes() []models.Note {
	notes := make([]models.Note, 0, len(mds.notes))
	for _, note := range mds.notes {
		notes = append(notes, note)
	}
	return notes
}

func (mds *MemoryDataStore) GetNote(id int64) (models.Note, bool) {
	note, exists := mds.notes[id]
	return note, exists
}

func (mds *MemoryDataStore) AddNote(note models.Note) error {
	mds.notes[note.ID] = note
	return nil
}

func (mds *MemoryDataStore) UpdateNote(id int64, note models.Note) error {
	mds.notes[id] = note
	return nil
}

func (mds *MemoryDataStore) DeleteNote(id int64) error {
	delete(mds.notes, id)
	return nil
}

// ID generation
func (mds *MemoryDataStore) NextID() int64 {
	id := mds.nextID
	mds.nextID++
	return id
}

// Happening operations
func (mds *MemoryDataStore) GetAllHappenings() []models.Happening {
	happenings := make([]models.Happening, 0, len(mds.happenings))
	for _, happening := range mds.happenings {
		happenings = append(happenings, happening)
	}
	return happenings
}

func (mds *MemoryDataStore) GetHappening(id int64) (models.Happening, bool) {
	happening, exists := mds.happenings[id]
	return happening, exists
}

func (mds *MemoryDataStore) AddHappening(happening models.Happening) error {
	mds.happenings[happening.ID] = happening
	return nil
}

func (mds *MemoryDataStore) UpdateHappening(id int64, happening models.Happening) error {
	mds.happenings[id] = happening
	return nil
}

func (mds *MemoryDataStore) DeleteHappening(id int64) error {
	delete(mds.happenings, id)
	return nil
}

// Persistence (no-op for memory store)
func (mds *MemoryDataStore) Save() error {
	return nil
}

// Factory functions using the new base store
func NewLogEntryService() storage.LogEntryService {
	dataStore := NewMemoryDataStore()
	return NewLogEntryBaseService(dataStore)
}

func NewLogNoteService() storage.LogNoteService {
	dataStore := NewMemoryDataStore()
	return NewLogNoteBaseService(dataStore)
}

func NewHappeningService() storage.HappeningService {
	dataStore := NewMemoryDataStore()
	return NewHappeningBaseService(dataStore)
}
