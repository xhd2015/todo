package memory

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

type MemoryStore struct {
	mu         sync.RWMutex
	logEntries map[int64]models.LogEntry
	notes      map[int64]models.Note
	nextID     int64
}

type LogEntryMemoryStore struct {
	*MemoryStore
}

type LogNoteMemoryStore struct {
	*MemoryStore
}

func New() *MemoryStore {
	return &MemoryStore{
		logEntries: make(map[int64]models.LogEntry),
		notes:      make(map[int64]models.Note),
		nextID:     1,
	}
}

func (ms *MemoryStore) nextIDValue() int64 {
	id := ms.nextID
	ms.nextID++
	return id
}

func NewLogEntryService() storage.LogEntryService {
	store := New()
	return &LogEntryMemoryStore{MemoryStore: store}
}

func NewLogNoteService() storage.LogNoteService {
	store := New()
	return &LogNoteMemoryStore{MemoryStore: store}
}

// LogEntry service methods
func (les *LogEntryMemoryStore) List(options storage.LogEntryListOptions) ([]models.LogEntry, int64, error) {
	les.mu.RLock()
	defer les.mu.RUnlock()

	var entries []models.LogEntry

	// Apply filter
	for _, entry := range les.logEntries {
		if options.Filter != "" {
			if !strings.Contains(strings.ToLower(entry.Text), strings.ToLower(options.Filter)) {
				continue
			}
		}

		// Handle history filtering
		if !options.IncludeHistory {
			// Filter out entries that are done and have done_time before today
			if entry.Done && entry.DoneTime != nil {
				today := time.Now().Truncate(24 * time.Hour)
				if entry.DoneTime.Before(today) {
					continue
				}
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

func (les *LogEntryMemoryStore) Add(entry models.LogEntry) (int64, error) {
	les.mu.Lock()
	defer les.mu.Unlock()

	entry.ID = les.nextIDValue()
	if entry.CreateTime.IsZero() {
		entry.CreateTime = time.Now()
	}
	if entry.UpdateTime.IsZero() {
		entry.UpdateTime = time.Now()
	}

	les.logEntries[entry.ID] = entry
	return entry.ID, nil
}

func (les *LogEntryMemoryStore) Delete(id int64) error {
	les.mu.Lock()
	defer les.mu.Unlock()

	if _, exists := les.logEntries[id]; !exists {
		return fmt.Errorf("log entry with id %d not found", id)
	}

	delete(les.logEntries, id)

	// Also delete all notes for this entry
	for noteID, note := range les.notes {
		if note.EntryID == id {
			delete(les.notes, noteID)
		}
	}

	return nil
}

func (les *LogEntryMemoryStore) Update(id int64, update models.LogEntryOptional) error {
	les.mu.Lock()
	defer les.mu.Unlock()

	entry, exists := les.logEntries[id]
	if !exists {
		return fmt.Errorf("log entry with id %d not found", id)
	}

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

	les.logEntries[id] = entry
	return nil
}

func (les *LogEntryMemoryStore) Move(id int64, newParentID int64) error {
	les.mu.Lock()
	defer les.mu.Unlock()

	entry, exists := les.logEntries[id]
	if !exists {
		return fmt.Errorf("log entry with id %d not found", id)
	}

	entry.ParentID = newParentID
	entry.UpdateTime = time.Now()
	les.logEntries[id] = entry
	return nil
}

// LogNote service methods
func (lns *LogNoteMemoryStore) List(entryID int64, options storage.LogNoteListOptions) ([]models.Note, int64, error) {
	lns.mu.RLock()
	defer lns.mu.RUnlock()

	var notes []models.Note

	// Apply filter
	for _, note := range lns.notes {
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

func (lns *LogNoteMemoryStore) ListForEntries(entryIDs []int64) (map[int64][]models.Note, error) {
	lns.mu.RLock()
	defer lns.mu.RUnlock()

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
	for _, note := range lns.notes {
		if entryIDSet[note.EntryID] {
			result[note.EntryID] = append(result[note.EntryID], note)
		}
	}

	return result, nil
}

func (lns *LogNoteMemoryStore) Add(entryID int64, note models.Note) (int64, error) {
	lns.mu.Lock()
	defer lns.mu.Unlock()

	// Check if entry exists
	if _, exists := lns.logEntries[entryID]; !exists {
		return 0, fmt.Errorf("log entry with id %d not found", entryID)
	}

	note.ID = lns.nextIDValue()
	note.EntryID = entryID
	if note.CreateTime.IsZero() {
		note.CreateTime = time.Now()
	}
	if note.UpdateTime.IsZero() {
		note.UpdateTime = time.Now()
	}

	lns.notes[note.ID] = note
	return note.ID, nil
}

func (lns *LogNoteMemoryStore) Delete(entryID int64, noteID int64) error {
	lns.mu.Lock()
	defer lns.mu.Unlock()

	note, exists := lns.notes[noteID]
	if !exists || note.EntryID != entryID {
		return fmt.Errorf("note with id %d not found for entry %d", noteID, entryID)
	}

	delete(lns.notes, noteID)
	return nil
}

func (lns *LogNoteMemoryStore) Update(entryID int64, noteID int64, update models.NoteOptional) error {
	lns.mu.Lock()
	defer lns.mu.Unlock()

	note, exists := lns.notes[noteID]
	if !exists || note.EntryID != entryID {
		return fmt.Errorf("note with id %d not found for entry %d", noteID, entryID)
	}

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

	lns.notes[noteID] = note
	return nil
}
