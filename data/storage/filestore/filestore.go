package filestore

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/data/storage/memory"
	"github.com/xhd2015/todo/models"
)

// FileDataStore implements DataStore interface for file-based storage
type FileDataStore struct {
	filePath string
	data     *FileData
}

type FileData struct {
	LogEntries []models.LogEntry `json:"log_entries"`
	Notes      []models.Note     `json:"notes"`
	NextID     int64             `json:"next_id"`
}

// NewFileDataStore creates a new file-based data store
func NewFileDataStore(filePath string) (*FileDataStore, error) {
	fds := &FileDataStore{
		filePath: filePath,
		data: &FileData{
			LogEntries: []models.LogEntry{},
			Notes:      []models.Note{},
			NextID:     1,
		},
	}

	// Try to load existing data
	if err := fds.load(); err != nil {
		// If file doesn't exist, that's ok, we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load file: %w", err)
		}
	}

	return fds, nil
}

func (fds *FileDataStore) load() error {
	data, err := os.ReadFile(fds.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, fds.data)
}

// Entry operations
func (fds *FileDataStore) GetAllEntries() []models.LogEntry {
	return fds.data.LogEntries
}

func (fds *FileDataStore) GetEntry(id int64) (models.LogEntry, bool) {
	for _, entry := range fds.data.LogEntries {
		if entry.ID == id {
			return entry, true
		}
	}
	return models.LogEntry{}, false
}

func (fds *FileDataStore) AddEntry(entry models.LogEntry) error {
	fds.data.LogEntries = append(fds.data.LogEntries, entry)
	return nil
}

func (fds *FileDataStore) UpdateEntry(id int64, entry models.LogEntry) error {
	for i, existingEntry := range fds.data.LogEntries {
		if existingEntry.ID == id {
			fds.data.LogEntries[i] = entry
			return nil
		}
	}
	return fmt.Errorf("entry with id %d not found", id)
}

func (fds *FileDataStore) DeleteEntry(id int64) error {
	for i, entry := range fds.data.LogEntries {
		if entry.ID == id {
			fds.data.LogEntries = append(fds.data.LogEntries[:i], fds.data.LogEntries[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("entry with id %d not found", id)
}

// Note operations
func (fds *FileDataStore) GetAllNotes() []models.Note {
	return fds.data.Notes
}

func (fds *FileDataStore) GetNote(id int64) (models.Note, bool) {
	for _, note := range fds.data.Notes {
		if note.ID == id {
			return note, true
		}
	}
	return models.Note{}, false
}

func (fds *FileDataStore) AddNote(note models.Note) error {
	fds.data.Notes = append(fds.data.Notes, note)
	return nil
}

func (fds *FileDataStore) UpdateNote(id int64, note models.Note) error {
	for i, existingNote := range fds.data.Notes {
		if existingNote.ID == id {
			fds.data.Notes[i] = note
			return nil
		}
	}
	return fmt.Errorf("note with id %d not found", id)
}

func (fds *FileDataStore) DeleteNote(id int64) error {
	for i, note := range fds.data.Notes {
		if note.ID == id {
			fds.data.Notes = append(fds.data.Notes[:i], fds.data.Notes[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("note with id %d not found", id)
}

// ID generation
func (fds *FileDataStore) NextID() int64 {
	id := fds.data.NextID
	fds.data.NextID++
	return id
}

// Persistence
func (fds *FileDataStore) Save() error {
	data, err := json.MarshalIndent(fds.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fds.filePath, data, 0644)
}

// Factory functions using the new base store
func NewLogEntryService(filePath string) (storage.LogEntryService, error) {
	dataStore, err := NewFileDataStore(filePath)
	if err != nil {
		return nil, err
	}
	return memory.NewLogEntryBaseService(dataStore), nil
}

func NewLogNoteService(filePath string) (storage.LogNoteService, error) {
	dataStore, err := NewFileDataStore(filePath)
	if err != nil {
		return nil, err
	}
	return memory.NewLogNoteBaseService(dataStore), nil
}
