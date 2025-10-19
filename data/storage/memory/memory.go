package memory

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

// MemoryDataStore implements DataStore interface for in-memory storage
type MemoryDataStore struct {
	logEntries   map[int64]models.LogEntry
	notes        map[int64]models.Note
	happenings   map[int64]models.Happening
	states       map[int64]models.State
	stateEvents  map[int64]models.StateEvent
	statesByName map[string]int64 // name -> state ID mapping
	nextID       int64
}

// NewMemoryDataStore creates a new in-memory data store
func NewMemoryDataStore() *MemoryDataStore {
	return &MemoryDataStore{
		logEntries:   make(map[int64]models.LogEntry),
		notes:        make(map[int64]models.Note),
		happenings:   make(map[int64]models.Happening),
		states:       make(map[int64]models.State),
		stateEvents:  make(map[int64]models.StateEvent),
		statesByName: make(map[string]int64),
		nextID:       1,
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

// State operations
func (mds *MemoryDataStore) GetAllStates() []models.State {
	states := make([]models.State, 0, len(mds.states))
	for _, state := range mds.states {
		states = append(states, state)
	}
	return states
}

func (mds *MemoryDataStore) GetState(id int64) (models.State, bool) {
	state, exists := mds.states[id]
	return state, exists
}

func (mds *MemoryDataStore) GetStateByName(name string) (models.State, bool) {
	if stateID, exists := mds.statesByName[name]; exists {
		return mds.GetState(stateID)
	}
	return models.State{}, false
}

func (mds *MemoryDataStore) AddState(state models.State) error {
	mds.states[state.ID] = state
	mds.statesByName[state.Name] = state.ID
	return nil
}

func (mds *MemoryDataStore) UpdateState(id int64, state models.State) error {
	if oldState, exists := mds.states[id]; exists {
		// Update name mapping if name changed
		if oldState.Name != state.Name {
			delete(mds.statesByName, oldState.Name)
			mds.statesByName[state.Name] = id
		}
	}
	mds.states[id] = state
	return nil
}

func (mds *MemoryDataStore) DeleteState(id int64) error {
	if state, exists := mds.states[id]; exists {
		delete(mds.statesByName, state.Name)
	}
	delete(mds.states, id)
	return nil
}

// StateEvent operations
func (mds *MemoryDataStore) GetAllStateEvents() []models.StateEvent {
	events := make([]models.StateEvent, 0, len(mds.stateEvents))
	for _, event := range mds.stateEvents {
		events = append(events, event)
	}
	return events
}

func (mds *MemoryDataStore) GetStateEvent(id int64) (models.StateEvent, bool) {
	event, exists := mds.stateEvents[id]
	return event, exists
}

func (mds *MemoryDataStore) AddStateEvent(event models.StateEvent) error {
	mds.stateEvents[event.ID] = event
	return nil
}

// MemoryStateRecordingService implements StateRecordingService for in-memory storage
type MemoryStateRecordingService struct {
	dataStore *MemoryDataStore
}

func NewStateRecordingService() storage.StateRecordingService {
	dataStore := NewMemoryDataStore()
	return &MemoryStateRecordingService{
		dataStore: dataStore,
	}
}

func (s *MemoryStateRecordingService) GetState(ctx context.Context, name string) (*models.State, error) {
	if state, exists := s.dataStore.GetStateByName(name); exists {
		return &state, nil
	}
	return nil, errors.New("state not found")
}

func (s *MemoryStateRecordingService) RecordStateEvent(ctx context.Context, name string, deltaScore float64) error {
	// Find the state by name
	state, exists := s.dataStore.GetStateByName(name)
	if !exists {
		return errors.New("state not found")
	}

	// Update the state score
	state.Score += deltaScore
	state.UpdateTime = time.Now()
	err := s.dataStore.UpdateState(state.ID, state)
	if err != nil {
		return err
	}

	// Create and add the state event
	eventID := s.dataStore.NextID()
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

	return s.dataStore.AddStateEvent(event)
}

func (s *MemoryStateRecordingService) CreateState(ctx context.Context, state *models.State) (*models.State, error) {
	if state == nil {
		return nil, errors.New("state cannot be nil")
	}

	// Check if state with same name already exists
	if _, exists := s.dataStore.GetStateByName(state.Name); exists {
		return nil, errors.New("state with this name already exists")
	}

	// Generate ID and set timestamps
	state.ID = s.dataStore.NextID()
	state.CreateTime = time.Now()
	state.UpdateTime = time.Now()

	err := s.dataStore.AddState(*state)
	if err != nil {
		return nil, err
	}

	return state, nil
}

func (s *MemoryStateRecordingService) ListStates(ctx context.Context, scope string) ([]*models.State, error) {
	allStates := s.dataStore.GetAllStates()
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

func (s *MemoryStateRecordingService) GetStateEvents(ctx context.Context, stateID int64, limit int) ([]*models.StateEvent, error) {
	allEvents := s.dataStore.GetAllStateEvents()
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
