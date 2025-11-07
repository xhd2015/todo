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

	// Happening operations
	GetAllHappenings() []models.Happening
	GetHappening(id int64) (models.Happening, bool)
	AddHappening(happening models.Happening) error
	UpdateHappening(id int64, happening models.Happening) error
	DeleteHappening(id int64) error

	// State operations
	GetAllStates() []models.State
	GetState(id int64) (models.State, bool)
	GetStateByName(name string) (models.State, bool)
	AddState(state models.State) error
	UpdateState(id int64, state models.State) error
	DeleteState(id int64) error

	// StateEvent operations
	GetAllStateEvents() []models.StateEvent
	GetStateEvent(id int64) (models.StateEvent, bool)
	AddStateEvent(event models.StateEvent) error

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

// HappeningBaseStore implements storage.HappeningService using BaseStore
type HappeningBaseStore struct {
	*BaseStore
}

// StateRecordingBaseStore implements storage.StateRecordingService using BaseStore
type StateRecordingBaseStore struct {
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

// NewHappeningBaseService creates a HappeningService using the given DataStore
func NewHappeningBaseService(data DataStore) storage.HappeningService {
	base := NewBaseStore(data)
	return &HappeningBaseStore{BaseStore: base}
}

// NewStateRecordingBaseService creates a StateRecordingService using the given DataStore
func NewStateRecordingBaseService(data DataStore) storage.StateRecordingService {
	base := NewBaseStore(data)
	return &StateRecordingBaseStore{BaseStore: base}
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
	if update.Collapsed != nil {
		entry.Collapsed = *update.Collapsed
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

// Happening service methods
func (hbs *HappeningBaseStore) List(options storage.HappeningListOptions) ([]*models.Happening, int64, error) {
	hbs.mu.RLock()
	defer hbs.mu.RUnlock()

	allHappenings := hbs.data.GetAllHappenings()
	var happenings []*models.Happening

	// Apply filter
	for _, happening := range allHappenings {
		if options.Filter != "" {
			if !strings.Contains(strings.ToLower(happening.Content), strings.ToLower(options.Filter)) {
				continue
			}
		}
		// Convert to pointer for consistency with interface
		happenings = append(happenings, &happening)
	}

	total := int64(len(happenings))

	// Apply sorting
	if options.SortBy != "" {
		sort.Slice(happenings, func(i, j int) bool {
			var less bool
			switch options.SortBy {
			case "id":
				less = happenings[i].ID < happenings[j].ID
			case "content":
				less = happenings[i].Content < happenings[j].Content
			case "create_time":
				less = happenings[i].CreateTime.Before(happenings[j].CreateTime)
			case "update_time":
				less = happenings[i].UpdateTime.Before(happenings[j].UpdateTime)
			default:
				less = happenings[i].ID < happenings[j].ID
			}

			if options.SortOrder == "desc" {
				return !less
			}
			return less
		})
	}

	// Apply pagination
	if options.Offset > 0 {
		if options.Offset >= len(happenings) {
			return []*models.Happening{}, total, nil
		}
		happenings = happenings[options.Offset:]
	}

	if options.Limit > 0 && options.Limit < len(happenings) {
		happenings = happenings[:options.Limit]
	}

	return happenings, total, nil
}

func (hbs *HappeningBaseStore) Add(ctx context.Context, happening *models.Happening) (*models.Happening, error) {
	if happening == nil {
		return nil, fmt.Errorf("happening cannot be nil")
	}
	if happening.Content == "" {
		return nil, fmt.Errorf("happening content cannot be empty")
	}

	hbs.mu.Lock()
	defer hbs.mu.Unlock()

	// Generate new ID and set timestamps
	newHappening := *happening
	newHappening.ID = hbs.data.NextID()
	now := time.Now()
	newHappening.CreateTime = now
	newHappening.UpdateTime = now

	// Add to data store
	if err := hbs.data.AddHappening(newHappening); err != nil {
		return nil, fmt.Errorf("failed to add happening: %w", err)
	}

	// Save data
	if err := hbs.data.Save(); err != nil {
		return nil, fmt.Errorf("failed to save data: %w", err)
	}

	return &newHappening, nil
}

func (hbs *HappeningBaseStore) Update(ctx context.Context, id int64, update *models.HappeningOptional) (*models.Happening, error) {
	if update == nil {
		return nil, fmt.Errorf("update cannot be nil")
	}

	hbs.mu.Lock()
	defer hbs.mu.Unlock()

	// Check if happening exists
	existing, exists := hbs.data.GetHappening(id)
	if !exists {
		return nil, fmt.Errorf("happening with id %d not found", id)
	}

	// Apply the optional updates to the existing happening
	updatedHappening := existing
	updatedHappening.Update(update)

	// Update in data store
	if err := hbs.data.UpdateHappening(id, updatedHappening); err != nil {
		return nil, fmt.Errorf("failed to update happening: %w", err)
	}

	// Save data
	if err := hbs.data.Save(); err != nil {
		return nil, fmt.Errorf("failed to save data: %w", err)
	}

	return &updatedHappening, nil
}

func (hbs *HappeningBaseStore) Delete(ctx context.Context, id int64) error {
	hbs.mu.Lock()
	defer hbs.mu.Unlock()

	// Check if happening exists
	if _, exists := hbs.data.GetHappening(id); !exists {
		return fmt.Errorf("happening with id %d not found", id)
	}

	// Delete from data store
	if err := hbs.data.DeleteHappening(id); err != nil {
		return fmt.Errorf("failed to delete happening: %w", err)
	}

	// Save data
	if err := hbs.data.Save(); err != nil {
		return fmt.Errorf("failed to save data: %w", err)
	}

	return nil
}

// StateRecordingService methods
func (srs *StateRecordingBaseStore) GetState(ctx context.Context, name string) (*models.State, error) {
	srs.mu.RLock()
	defer srs.mu.RUnlock()

	if state, exists := srs.data.GetStateByName(name); exists {
		return &state, nil
	}
	return nil, fmt.Errorf("state not found")
}

func (srs *StateRecordingBaseStore) RecordStateEvent(ctx context.Context, name string, deltaScore float64) error {
	srs.mu.Lock()
	defer srs.mu.Unlock()

	// Find the state by name
	state, exists := srs.data.GetStateByName(name)
	if !exists {
		return fmt.Errorf("state not found")
	}

	// Update the state score
	state.Score += deltaScore
	state.UpdateTime = time.Now()
	err := srs.data.UpdateState(state.ID, state)
	if err != nil {
		return err
	}

	// Create and add the state event
	eventID := srs.data.NextID()
	event := models.StateEvent{
		ID:            eventID,
		StateRecordID: state.ID,
		RecordData:    "",
		DeltaScore:    deltaScore,
		Description:   "",
		Details:       "",
		Scope:         state.Scope,
		CreateTime:    time.Now(),
		UpdateTime:    time.Now(),
	}

	if err := srs.data.AddStateEvent(event); err != nil {
		return err
	}

	return srs.data.Save()
}

func (srs *StateRecordingBaseStore) CreateState(ctx context.Context, state *models.State) (*models.State, error) {
	if state == nil {
		return nil, fmt.Errorf("state cannot be nil")
	}

	srs.mu.Lock()
	defer srs.mu.Unlock()

	// Check if state with same name already exists
	if _, exists := srs.data.GetStateByName(state.Name); exists {
		return nil, fmt.Errorf("state with this name already exists")
	}

	// Generate ID and set timestamps
	state.ID = srs.data.NextID()
	state.CreateTime = time.Now()
	state.UpdateTime = time.Now()

	err := srs.data.AddState(*state)
	if err != nil {
		return nil, err
	}

	if err := srs.data.Save(); err != nil {
		return nil, err
	}

	return state, nil
}

func (srs *StateRecordingBaseStore) ListStates(ctx context.Context, scope string) ([]*models.State, error) {
	srs.mu.RLock()
	defer srs.mu.RUnlock()

	allStates := srs.data.GetAllStates()
	var filteredStates []*models.State

	for _, state := range allStates {
		// Filter by scope if provided
		if scope == "" || strings.Contains(state.Scope, scope) {
			stateCopy := state
			filteredStates = append(filteredStates, &stateCopy)
		}
	}

	return filteredStates, nil
}

func (srs *StateRecordingBaseStore) GetStateEvents(ctx context.Context, stateID int64, limit int) ([]*models.StateEvent, error) {
	srs.mu.RLock()
	defer srs.mu.RUnlock()

	allEvents := srs.data.GetAllStateEvents()
	var filteredEvents []*models.StateEvent

	for _, event := range allEvents {
		if event.StateRecordID == stateID {
			eventCopy := event
			filteredEvents = append(filteredEvents, &eventCopy)
		}
	}

	// Apply limit if specified
	if limit > 0 && len(filteredEvents) > limit {
		filteredEvents = filteredEvents[:limit]
	}

	return filteredEvents, nil
}

func (srs *StateRecordingBaseStore) GetStateHistory(ctx context.Context, options storage.GetStateHistoryOptions) ([]models.StateHistoryPoint, error) {
	srs.mu.RLock()
	defer srs.mu.RUnlock()

	// Set default days
	days := options.Days
	if days <= 0 {
		days = 30
	}

	// Find state IDs based on names filter
	var stateIDs []int64
	if len(options.Names) == 0 {
		// No filter - get all states
		allStates := srs.data.GetAllStates()
		for _, state := range allStates {
			stateIDs = append(stateIDs, state.ID)
		}
	} else {
		// Filter by names
		for _, name := range options.Names {
			state, exists := srs.data.GetStateByName(name)
			if exists {
				stateIDs = append(stateIDs, state.ID)
			}
		}
	}

	// Guard clause: check if any states found
	if len(stateIDs) == 0 {
		return []models.StateHistoryPoint{}, nil
	}

	// Calculate start time (N days ago)
	startTime := time.Now().AddDate(0, 0, -days)

	// Get all events
	allEvents := srs.data.GetAllStateEvents()

	// Group events by date and calculate cumulative score
	dateScores := make(map[string]float64)
	var dates []string
	var cumulativeScore float64

	// Create a map for quick state ID lookup
	stateIDMap := make(map[int64]bool)
	for _, id := range stateIDs {
		stateIDMap[id] = true
	}

	for _, event := range allEvents {
		if !stateIDMap[event.StateRecordID] {
			continue
		}
		if event.CreateTime.Before(startTime) {
			continue
		}

		dateStr := event.CreateTime.Format("2006-01-02")
		cumulativeScore += event.DeltaScore

		// Track dates in order if not seen before
		if _, exists := dateScores[dateStr]; !exists {
			dates = append(dates, dateStr)
		}
		dateScores[dateStr] = cumulativeScore
	}

	// Build history response
	history := make([]models.StateHistoryPoint, 0, len(dates))
	for _, date := range dates {
		history = append(history, models.StateHistoryPoint{
			Date:  date,
			Score: dateScores[date],
		})
	}

	return history, nil
}
