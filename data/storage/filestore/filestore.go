package filestore

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

type FileStore struct {
	filePath string
	mu       sync.RWMutex
	data     *FileData
}

type LogEntryFileStore struct {
	*FileStore
}

type LogNoteFileStore struct {
	*FileStore
}

type FileData struct {
	LogEntries []models.LogEntry `json:"log_entries"`
	Notes      []models.Note     `json:"notes"`
	NextID     int64             `json:"next_id"`
}

func New(filePath string) (*FileStore, error) {
	fs := &FileStore{
		filePath: filePath,
		data: &FileData{
			LogEntries: []models.LogEntry{},
			Notes:      []models.Note{},
			NextID:     1,
		},
	}

	// Try to load existing data
	if err := fs.load(); err != nil {
		// If file doesn't exist, that's ok, we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load file: %w", err)
		}
	}

	return fs, nil
}

func NewLogEntryService(filePath string) (storage.LogEntryService, error) {
	fs, err := New(filePath)
	if err != nil {
		return nil, err
	}
	return &LogEntryFileStore{FileStore: fs}, nil
}

func NewLogNoteService(filePath string) (storage.LogNoteService, error) {
	fs, err := New(filePath)
	if err != nil {
		return nil, err
	}
	return &LogNoteFileStore{FileStore: fs}, nil
}

func (fs *FileStore) load() error {
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, fs.data)
}

func (fs *FileStore) save() error {
	data, err := json.MarshalIndent(fs.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.filePath, data, 0644)
}

func (fs *FileStore) nextID() int64 {
	id := fs.data.NextID
	fs.data.NextID++
	return id
}

// LogEntry service methods
func (les *LogEntryFileStore) List(options storage.LogEntryListOptions) ([]models.LogEntry, int64, error) {
	fs := les.FileStore
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries := make([]models.LogEntry, 0, len(fs.data.LogEntries))

	// Apply filter
	for _, entry := range fs.data.LogEntries {
		if options.Filter != "" {
			if !strings.Contains(strings.ToLower(entry.Text), strings.ToLower(options.Filter)) {
				continue
			}
		}
		entries = append(entries, entry)
	}

	total := int64(len(entries))

	// Apply sorting
	if options.SortBy != "" {
		sort.Slice(entries, func(i, j int) bool {
			var less bool
			switch options.SortBy {
			case "id":
				less = entries[i].ID < entries[j].ID
			case "text":
				less = entries[i].Text < entries[j].Text
			case "done":
				less = !entries[i].Done && entries[j].Done
			case "create_time":
				// If AdjustedTopTime is set, use it for sorting priority
				if entries[i].AdjustedTopTime != 0 || entries[j].AdjustedTopTime != 0 {
					less = entries[i].AdjustedTopTime < entries[j].AdjustedTopTime
				} else {
					less = entries[i].CreateTime.Before(entries[j].CreateTime)
				}
			case "update_time":
				less = entries[i].UpdateTime.Before(entries[j].UpdateTime)
			default:
				less = entries[i].ID < entries[j].ID
			}

			if options.SortOrder == "desc" {
				return !less
			}
			return less
		})
	}

	// Apply pagination
	if options.Offset > 0 {
		if options.Offset >= len(entries) {
			return []models.LogEntry{}, total, nil
		}
		entries = entries[options.Offset:]
	}

	if options.Limit > 0 && options.Limit < len(entries) {
		entries = entries[:options.Limit]
	}

	return entries, total, nil
}

func (les *LogEntryFileStore) Add(entry models.LogEntry) (int64, error) {
	fs := les.FileStore
	fs.mu.Lock()
	defer fs.mu.Unlock()

	entry.ID = fs.nextID()
	if entry.CreateTime.IsZero() {
		entry.CreateTime = time.Now()
	}
	if entry.UpdateTime.IsZero() {
		entry.UpdateTime = time.Now()
	}

	fs.data.LogEntries = append(fs.data.LogEntries, entry)

	if err := fs.save(); err != nil {
		return 0, err
	}

	return entry.ID, nil
}

func (les *LogEntryFileStore) Delete(id int64) error {
	fs := les.FileStore
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for i, entry := range fs.data.LogEntries {
		if entry.ID == id {
			fs.data.LogEntries = append(fs.data.LogEntries[:i], fs.data.LogEntries[i+1:]...)

			// Also delete all notes for this entry
			var filteredNotes []models.Note
			for _, note := range fs.data.Notes {
				if note.EntryID != id {
					filteredNotes = append(filteredNotes, note)
				}
			}
			fs.data.Notes = filteredNotes

			return fs.save()
		}
	}

	return fmt.Errorf("log entry with id %d not found", id)
}

func (les *LogEntryFileStore) Update(id int64, update models.LogEntryOptional) error {
	fs := les.FileStore
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for i, entry := range fs.data.LogEntries {
		if entry.ID == id {
			if update.Text != nil {
				entry.Text = *update.Text
			}
			if update.Done != nil {
				entry.Done = *update.Done
			}
			if update.DoneTime != nil {
				entry.DoneTime = *update.DoneTime
			}
			if update.CreateTime != nil {
				entry.CreateTime = *update.CreateTime
			}
			if update.UpdateTime != nil {
				entry.UpdateTime = *update.UpdateTime
			} else {
				entry.UpdateTime = time.Now()
			}
			if update.AdjustedTopTime != nil {
				entry.AdjustedTopTime = *update.AdjustedTopTime
			}
			if update.HighlightLevel != nil {
				entry.HighlightLevel = *update.HighlightLevel
			}
			if update.ParentID != nil {
				entry.ParentID = *update.ParentID
			}

			fs.data.LogEntries[i] = entry
			return fs.save()
		}
	}

	return fmt.Errorf("log entry with id %d not found", id)
}

func (les *LogEntryFileStore) Move(id int64, newParentID int64) error {
	fs := les.FileStore
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for i, entry := range fs.data.LogEntries {
		if entry.ID == id {
			entry.ParentID = newParentID
			entry.UpdateTime = time.Now()
			fs.data.LogEntries[i] = entry
			return fs.save()
		}
	}

	return fmt.Errorf("log entry with id %d not found", id)
}

// LogNote service methods
func (lns *LogNoteFileStore) List(entryID int64, options storage.LogNoteListOptions) ([]models.Note, int64, error) {
	fs := lns.FileStore
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	notes := make([]models.Note, 0)

	// Apply filter
	for _, note := range fs.data.Notes {
		if note.EntryID != entryID {
			continue
		}
		if options.Filter != "" {
			if !strings.Contains(strings.ToLower(note.Text), strings.ToLower(options.Filter)) {
				continue
			}
		}
		notes = append(notes, note)
	}

	total := int64(len(notes))

	// Apply sorting
	if options.SortBy != "" {
		sort.Slice(notes, func(i, j int) bool {
			var less bool
			switch options.SortBy {
			case "id":
				less = notes[i].ID < notes[j].ID
			case "text":
				less = notes[i].Text < notes[j].Text
			case "create_time":
				less = notes[i].CreateTime.Before(notes[j].CreateTime)
			case "update_time":
				less = notes[i].UpdateTime.Before(notes[j].UpdateTime)
			default:
				less = notes[i].ID < notes[j].ID
			}

			if options.SortOrder == "desc" {
				return !less
			}
			return less
		})
	}

	// Apply pagination
	if options.Offset > 0 {
		if options.Offset >= len(notes) {
			return []models.Note{}, total, nil
		}
		notes = notes[options.Offset:]
	}

	if options.Limit > 0 && options.Limit < len(notes) {
		notes = notes[:options.Limit]
	}

	return notes, total, nil
}

func (lns *LogNoteFileStore) ListForEntries(entryIDs []int64) (map[int64][]models.Note, error) {
	fs := lns.FileStore
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	result := make(map[int64][]models.Note)

	// Initialize empty slices for all requested entry IDs
	for _, entryID := range entryIDs {
		result[entryID] = []models.Note{}
	}

	// Create a set for faster lookup
	entryIDSet := make(map[int64]bool)
	for _, entryID := range entryIDs {
		entryIDSet[entryID] = true
	}

	// Collect notes for requested entries
	for _, note := range fs.data.Notes {
		if entryIDSet[note.EntryID] {
			result[note.EntryID] = append(result[note.EntryID], note)
		}
	}

	return result, nil
}

func (lns *LogNoteFileStore) Add(entryID int64, note models.Note) (int64, error) {
	fs := lns.FileStore
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Check if entry exists
	entryExists := false
	for _, entry := range fs.data.LogEntries {
		if entry.ID == entryID {
			entryExists = true
			break
		}
	}

	if !entryExists {
		return 0, fmt.Errorf("log entry with id %d not found", entryID)
	}

	note.ID = fs.nextID()
	note.EntryID = entryID
	if note.CreateTime.IsZero() {
		note.CreateTime = time.Now()
	}
	if note.UpdateTime.IsZero() {
		note.UpdateTime = time.Now()
	}

	fs.data.Notes = append(fs.data.Notes, note)

	if err := fs.save(); err != nil {
		return 0, err
	}

	return note.ID, nil
}

func (lns *LogNoteFileStore) Delete(entryID int64, noteID int64) error {
	fs := lns.FileStore
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for i, note := range fs.data.Notes {
		if note.ID == noteID && note.EntryID == entryID {
			fs.data.Notes = append(fs.data.Notes[:i], fs.data.Notes[i+1:]...)
			return fs.save()
		}
	}

	return fmt.Errorf("note with id %d not found for entry %d", noteID, entryID)
}

func (lns *LogNoteFileStore) Update(entryID int64, noteID int64, update models.NoteOptional) error {
	fs := lns.FileStore
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for i, note := range fs.data.Notes {
		if note.ID == noteID && note.EntryID == entryID {
			if update.Text != nil {
				note.Text = *update.Text
			}
			if update.CreateTime != nil {
				note.CreateTime = *update.CreateTime
			}
			if update.UpdateTime != nil {
				note.UpdateTime = *update.UpdateTime
			} else {
				note.UpdateTime = time.Now()
			}

			fs.data.Notes[i] = note
			return fs.save()
		}
	}

	return fmt.Errorf("note with id %d not found for entry %d", noteID, entryID)
}
