package data

import (
	"context"
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
	entries, _, err := svc.List(storage.LogEntryListOptions{
		IncludeHistory: showHistory,
	})
	if err != nil {
		return nil, err
	}

	// No need for manual filtering anymore - the storage layer handles it
	filteredEntries := entries

	var entriesView []*models.LogEntryView
	// Create a map for quick lookup
	entryMap := make(map[int64]*models.LogEntryView)

	// Collect all entry IDs for batch note loading
	entryIDs := make([]int64, 0, len(filteredEntries))
	for _, entry := range filteredEntries {
		entryIDs = append(entryIDs, entry.ID)
	}

	// Batch load all notes for all entries
	allNotes, err := noteSvc.ListForEntries(entryIDs)
	if err != nil {
		return nil, err
	}

	for _, entry := range filteredEntries {
		notes := allNotes[entry.ID] // Get notes for this entry
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

	// Build parent-child relationships and filter root entries
	var rootEntries []*models.LogEntryView
	for _, entryView := range entriesView {
		if entryView.Data.ParentID != 0 {
			if parent, exists := entryMap[entryView.Data.ParentID]; exists {
				parent.Children = append(parent.Children, entryView)
			}
		} else {
			// Only add root entries (ParentID == 0) to the result
			rootEntries = append(rootEntries, entryView)
		}
	}

	sortEntries(rootEntries)
	return rootEntries, nil
}

// Init initializes with default behavior (no history)
func (m *LogManager) Init() error {
	return m.InitWithHistory(false)
}

func sortEntries(entries []*models.LogEntryView) {
	flatSortEntries(entries)
	for _, entry := range entries {
		sortEntries(entry.Children)
	}
}

func flatSortEntries(entries []*models.LogEntryView) {
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
		var findAndAddToParent func(entries []*models.LogEntryView) bool
		findAndAddToParent = func(entries []*models.LogEntryView) bool {
			for _, existingEntry := range entries {
				if existingEntry.Data.ID == entry.ParentID {
					existingEntry.Children = append(existingEntry.Children, entryView)
					return true
				}
				if findAndAddToParent(existingEntry.Children) {
					return true
				}
			}
			return false
		}
		findAndAddToParent(m.Entries)
	} else {
		// Only add root entries (ParentID == 0) to the top-level entries
		m.Entries = append(m.Entries, entryView)
	}

	return id, nil
}

func (m *LogManager) Get(id int64) (*models.LogEntryView, error) {
	var result *models.LogEntryView
	var traverse func(entries []*models.LogEntryView) bool
	traverse = func(entries []*models.LogEntryView) bool {
		for _, e := range entries {
			if e.Data.ID == id {
				result = e
				return true
			}
			if traverse(e.Children) {
				return true
			}
		}
		return false
	}
	traverse(m.Entries)
	if result == nil {
		return nil, fmt.Errorf("entry with id %d not found", id)
	}
	return result, nil
}

func (m *LogManager) Update(id int64, entry models.LogEntryOptional) error {
	if entry.UpdateTime == nil {
		t := time.Now()
		entry.UpdateTime = &t
	}

	targetEntry, err := m.Get(id)
	if err != nil {
		return err
	}
	oldParentID := targetEntry.Data.ParentID

	err = m.LogEntryService.Update(id, entry)
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

	m.deleteEntry(id)
	return nil
}

func (m *LogManager) deleteEntry(id int64) *models.LogEntryView {
	// bread-first search
	var foundEntry *models.LogEntryView
	var traverse func(entries []*models.LogEntryView) ([]*models.LogEntryView, bool)
	traverse = func(entries []*models.LogEntryView) ([]*models.LogEntryView, bool) {
		for i, e := range entries {
			if e.Data.ID == id {
				newEntries := make([]*models.LogEntryView, len(entries)-1)
				copy(newEntries, entries[:i])
				copy(newEntries[i:], entries[i+1:])
				foundEntry = e
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
	return foundEntry
}

func (m *LogManager) AddNote(entryID int64, note models.Note) error {
	entry, err := m.Get(entryID)
	if err != nil {
		return err
	}

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

	entry.Notes = append(entry.Notes, &models.NoteView{
		Data: &note,
	})

	return nil
}

func (m *LogManager) DeleteNote(entryID int64, noteID int64) error {
	entry, err := m.Get(entryID)
	if err != nil {
		return err
	}

	err = m.LogNoteService.Delete(entryID, noteID)
	if err != nil {
		return err
	}

	for i, n := range entry.Notes {
		if n.Data.ID == noteID {
			entry.Notes = append(entry.Notes[:i], entry.Notes[i+1:]...)
			break
		}
	}
	return nil
}

func (m *LogManager) UpdateNote(entryID int64, noteID int64, note models.NoteOptional) error {
	entry, err := m.Get(entryID)
	if err != nil {
		return err
	}

	err = m.LogNoteService.Update(entryID, noteID, note)
	if err != nil {
		return err
	}

	for _, n := range entry.Notes {
		if n.Data.ID == noteID {
			n.Data.Update(&note)
			return nil
		}
	}
	return nil
}

func (m *LogManager) Move(id int64, newParentID int64) error {
	err := m.LogEntryService.Move(id, newParentID)
	if err != nil {
		return err
	}

	// first, remove from old parent
	moved := m.deleteEntry(id)
	if moved == nil {
		return nil
	}

	// then, add to new parent
	var traverse func(entry *models.LogEntryView) bool
	traverse = func(entry *models.LogEntryView) bool {
		if entry.Data.ID == newParentID {
			moved.Data.ParentID = newParentID
			moved.Data.UpdateTime = time.Now()
			entry.Children = append(entry.Children, moved)
			flatSortEntries(entry.Children)
			return true
		}
		for _, child := range entry.Children {
			if traverse(child) {
				return true
			}
		}
		return false
	}
	for _, entry := range m.Entries {
		if traverse(entry) {
			break
		}
	}

	return nil
}

// LoadAll loads all descendants of a given root ID, including history entries
// Returns a single LogEntryView containing all children including history children
func (m *LogManager) LoadAll(ctx context.Context, rootID int64) (*models.LogEntryView, error) {
	// Load all entries for the root and its descendants
	entries, err := m.LogEntryService.LoadAll(rootID)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("entry with id %d not found", rootID)
	}

	// Collect all entry IDs for batch note loading
	entryIDs := make([]int64, 0, len(entries))
	for _, entry := range entries {
		entryIDs = append(entryIDs, entry.ID)
	}

	// Batch load all notes for all entries
	allNotes, err := m.LogNoteService.ListForEntries(entryIDs)
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookup
	entryMap := make(map[int64]*models.LogEntryView)

	// Convert entries to LogEntryView
	for _, entry := range entries {
		notes := allNotes[entry.ID] // Get notes for this entry
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
			ChildrenVisible: false, // Default to not visible
		}
		entryMap[entry.ID] = entryView
	}

	// Build parent-child relationships
	var rootEntry *models.LogEntryView
	for _, entryView := range entryMap {
		if entryView.Data.ID == rootID {
			rootEntry = entryView
		}
		if entryView.Data.ParentID != 0 {
			if parent, exists := entryMap[entryView.Data.ParentID]; exists {
				parent.Children = append(parent.Children, entryView)
			}
		}
	}

	if rootEntry == nil {
		return nil, fmt.Errorf("root entry with id %d not found", rootID)
	}

	// Sort children recursively
	sortEntries([]*models.LogEntryView{rootEntry})

	return rootEntry, nil
}
