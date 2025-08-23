package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

// DataStore defines the interface for the underlying data storage
type DataStore interface {
	// Entry operations
	GetAllEntries() []models.LogEntry
	GetEntry(id int64) (models.LogEntry, bool)
	AddEntry(entry models.LogEntry) error
	UpdateEntry(id int64, entry models.LogEntry) error
	DeleteEntry(id int64) error

	// Note operations
	GetAllNotes() []models.Note
	GetNote(id int64) (models.Note, bool)
	AddNote(note models.Note) error
	UpdateNote(id int64, note models.Note) error
	DeleteNote(id int64) error

	// ID generation
	NextID() int64

	// Persistence (for file-based stores)
	Save() error
}

// BaseStore provides common implementation for LogEntry and LogNote services
type BaseStore struct {
	mu   sync.RWMutex
	data DataStore
}

// NewBaseStore creates a new BaseStore with the given DataStore
func NewBaseStore(data DataStore) *BaseStore {
	return &BaseStore{
		data: data,
	}
}

// LogEntryBaseStore implements storage.LogEntryService using BaseStore
type LogEntryBaseStore struct {
	*BaseStore
}

// LogNoteBaseStore implements storage.LogNoteService using BaseStore
type LogNoteBaseStore struct {
	*BaseStore
}

// NewLogEntryBaseService creates a LogEntryService using the given DataStore
func NewLogEntryBaseService(data DataStore) storage.LogEntryService {
	base := NewBaseStore(data)
	return &LogEntryBaseStore{BaseStore: base}
}

// NewLogNoteBaseService creates a LogNoteService using the given DataStore
func NewLogNoteBaseService(data DataStore) storage.LogNoteService {
	base := NewBaseStore(data)
	return &LogNoteBaseStore{BaseStore: base}
}

// LogEntry service methods
func (les *LogEntryBaseStore) List(options storage.LogEntryListOptions) ([]models.LogEntry, int64, error) {
	les.mu.RLock()
	defer les.mu.RUnlock()

	allEntries := les.data.GetAllEntries()
	var entries []models.LogEntry

	// Apply filter
	for _, entry := range allEntries {
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

func (les *LogEntryBaseStore) Add(entry models.LogEntry) (int64, error) {
	les.mu.Lock()
	defer les.mu.Unlock()

	entry.ID = les.data.NextID()
	if entry.CreateTime.IsZero() {
		entry.CreateTime = time.Now()
	}
	if entry.UpdateTime.IsZero() {
		entry.UpdateTime = time.Now()
	}

	if err := les.data.AddEntry(entry); err != nil {
		return 0, err
	}

	if err := les.data.Save(); err != nil {
		return 0, err
	}

	return entry.ID, nil
}

func (les *LogEntryBaseStore) Delete(id int64) error {
	les.mu.Lock()
	defer les.mu.Unlock()

	if _, exists := les.data.GetEntry(id); !exists {
		return fmt.Errorf("log entry with id %d not found", id)
	}

	if err := les.data.DeleteEntry(id); err != nil {
		return err
	}

	// Also delete all notes for this entry
	allNotes := les.data.GetAllNotes()
	for _, note := range allNotes {
		if note.EntryID == id {
			if err := les.data.DeleteNote(note.ID); err != nil {
				return err
			}
		}
	}

	return les.data.Save()
}

func (les *LogEntryBaseStore) Update(id int64, update models.LogEntryOptional) error {
	les.mu.Lock()
	defer les.mu.Unlock()

	entry, exists := les.data.GetEntry(id)
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

	if err := les.data.UpdateEntry(id, entry); err != nil {
		return err
	}

	return les.data.Save()
}

func (les *LogEntryBaseStore) Move(id int64, newParentID int64) error {
	les.mu.Lock()
	defer les.mu.Unlock()

	entry, exists := les.data.GetEntry(id)
	if !exists {
		return fmt.Errorf("log entry with id %d not found", id)
	}

	entry.ParentID = newParentID
	entry.UpdateTime = time.Now()

	if err := les.data.UpdateEntry(id, entry); err != nil {
		return err
	}

	return les.data.Save()
}

func (les *LogEntryBaseStore) GetTree(ctx context.Context, id int64, includeHistory bool) ([]models.LogEntry, error) {
	les.mu.RLock()
	defer les.mu.RUnlock()

	// Find all descendants of the root entry using a recursive approach
	var result []models.LogEntry
	allEntries := les.data.GetAllEntries()

	// Create a map for quick lookup
	entryMap := make(map[int64]models.LogEntry)
	for _, entry := range allEntries {
		entryMap[entry.ID] = entry
	}

	// Find the root entry first
	rootEntry, exists := entryMap[id]
	if !exists {
		return nil, fmt.Errorf("root entry with id %d not found", id)
	}

	// Recursive function to collect all descendants
	var collectDescendants func(parentID int64)
	collectDescendants = func(parentID int64) {
		for _, entry := range allEntries {
			if entry.ParentID == parentID {
				// Apply history filter if needed
				if !includeHistory && entry.Done && entry.DoneTime != nil {
					// Skip done entries if not including history
					continue
				}
				result = append(result, entry)
				collectDescendants(entry.ID)
			}
		}
	}

	// Add root entry first
	result = append(result, rootEntry)

	// Collect all descendants
	collectDescendants(id)

	return result, nil
}

// LogNote service methods
func (lns *LogNoteBaseStore) List(entryID int64, options storage.LogNoteListOptions) ([]models.Note, int64, error) {
	lns.mu.RLock()
	defer lns.mu.RUnlock()

	allNotes := lns.data.GetAllNotes()
	var notes []models.Note

	// Apply filter
	for _, note := range allNotes {
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

func (lns *LogNoteBaseStore) ListForEntries(entryIDs []int64) (map[int64][]models.Note, error) {
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
	allNotes := lns.data.GetAllNotes()
	for _, note := range allNotes {
		if entryIDSet[note.EntryID] {
			result[note.EntryID] = append(result[note.EntryID], note)
		}
	}

	return result, nil
}

func (lns *LogNoteBaseStore) Add(entryID int64, note models.Note) (int64, error) {
	lns.mu.Lock()
	defer lns.mu.Unlock()

	// Check if entry exists
	if _, exists := lns.data.GetEntry(entryID); !exists {
		return 0, fmt.Errorf("log entry with id %d not found", entryID)
	}

	note.ID = lns.data.NextID()
	note.EntryID = entryID
	if note.CreateTime.IsZero() {
		note.CreateTime = time.Now()
	}
	if note.UpdateTime.IsZero() {
		note.UpdateTime = time.Now()
	}

	if err := lns.data.AddNote(note); err != nil {
		return 0, err
	}

	if err := lns.data.Save(); err != nil {
		return 0, err
	}

	return note.ID, nil
}

func (lns *LogNoteBaseStore) Delete(entryID int64, noteID int64) error {
	lns.mu.Lock()
	defer lns.mu.Unlock()

	note, exists := lns.data.GetNote(noteID)
	if !exists || note.EntryID != entryID {
		return fmt.Errorf("note with id %d not found for entry %d", noteID, entryID)
	}

	if err := lns.data.DeleteNote(noteID); err != nil {
		return err
	}

	return lns.data.Save()
}

func (lns *LogNoteBaseStore) Update(entryID int64, noteID int64, update models.NoteOptional) error {
	lns.mu.Lock()
	defer lns.mu.Unlock()

	note, exists := lns.data.GetNote(noteID)
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

	if err := lns.data.UpdateNote(noteID, note); err != nil {
		return err
	}

	return lns.data.Save()
}
