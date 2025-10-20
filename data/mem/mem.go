package mem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/xhd2015/todo/log"
)

type IDAware interface {
	GetID() int64
	SetID(id int64)
}

type Options struct {
	Limit  int
	Offset int
}

type JSONStorage struct {
	Data     []json.RawMessage `json:"data"`
	NextID   int64             `json:"next_id"`
	Metadata map[string]any    `json:"metadata,omitempty"`
}

type MemStore[T IDAware] struct {
	mu       sync.RWMutex
	filePath string
	storage  JSONStorage
	items    []T // In-memory cache

	// Background save optimization
	dirty     bool
	saveMu    sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	saveDelay time.Duration
}

// NewMemStore creates a new MemStore with the specified file path
func NewMemStore[T IDAware](filePath string) (*MemStore[T], error) {
	ctx, cancel := context.WithCancel(context.Background())

	store := &MemStore[T]{
		filePath: filePath,
		storage: JSONStorage{
			Data:     make([]json.RawMessage, 0),
			NextID:   1,
			Metadata: make(map[string]any),
		},
		items:     nil,
		ctx:       ctx,
		cancel:    cancel,
		saveDelay: 500 * time.Millisecond, // Default 500ms delay for batching
	}

	// Load existing data if file exists
	if err := store.load(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	// Start background save goroutine
	go store.backgroundSaver()

	return store, nil
}

// load reads the JSON file and populates the in-memory cache
func (ms *MemStore[T]) load() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(ms.filePath); os.IsNotExist(err) {
		// File doesn't exist, start with empty storage
		log.Infof(context.Background(), "file does not exist, starting with empty storage")
		return nil
	}

	// Read file
	data, err := os.ReadFile(ms.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, &ms.storage); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Populate in-memory cache
	ms.items = nil
	for _, rawItem := range ms.storage.Data {
		var item T
		if err := json.Unmarshal(rawItem, &item); err != nil {
			log.Errorf(context.Background(), "failed to unmarshal item: %v", err)
			continue // Skip invalid items
		}
		ms.items = append(ms.items, item)
	}

	log.Infof(context.Background(), "loaded %d items", len(ms.items))

	return nil
}

// backgroundSaver runs in a background goroutine and periodically saves dirty data
func (ms *MemStore[T]) backgroundSaver() {
	ticker := time.NewTicker(ms.saveDelay)
	defer ticker.Stop()

	for {
		select {
		case <-ms.ctx.Done():
			// Final save before shutdown
			ms.saveMu.Lock()
			if ms.dirty {
				ms.saveImmediate()
			}
			ms.saveMu.Unlock()
			return
		case <-ticker.C:
			ms.saveMu.Lock()
			if ms.dirty {
				ms.saveImmediate()
				ms.dirty = false
			}
			ms.saveMu.Unlock()
		}
	}
}

// markDirty marks the store as needing a save
func (ms *MemStore[T]) markDirty() {
	ms.saveMu.Lock()
	ms.dirty = true
	ms.saveMu.Unlock()
}

// Close gracefully shuts down the MemStore and ensures final save
func (ms *MemStore[T]) Close() error {
	ms.cancel()
	// Give background goroutine time to complete final save
	time.Sleep(10 * time.Millisecond)
	return nil
}

// saveImmediate writes the current state to the JSON file immediately
func (ms *MemStore[T]) saveImmediate() error {
	// Update storage data from in-memory cache
	ms.storage.Data = make([]json.RawMessage, 0, len(ms.items))
	for _, item := range ms.items {
		data, err := json.Marshal(item)
		if err != nil {
			continue // Skip items that can't be marshaled
		}
		ms.storage.Data = append(ms.storage.Data, json.RawMessage(data))
	}

	// Marshal entire storage
	data, err := json.MarshalIndent(ms.storage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(ms.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// List returns a slice of items with pagination support
func (ms *MemStore[T]) List(options Options) ([]T, int64, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Convert map to slice
	allItems := ms.items

	total := int64(len(allItems))

	// Apply offset
	if options.Offset > 0 {
		if options.Offset >= len(allItems) {
			return []T{}, total, nil
		}
		allItems = allItems[options.Offset:]
	}

	// Apply limit
	if options.Limit > 0 && len(allItems) > options.Limit {
		allItems = allItems[:options.Limit]
	}

	return allItems, total, nil
}

// Get retrieves an item by ID
func (ms *MemStore[T]) Get(id int64) (T, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, item := range ms.items {
		if item.GetID() == id {
			return item, nil
		}
	}

	var zero T
	return zero, fmt.Errorf("item with id %d not found", id)
}

// Update updates an existing item or creates a new one
func (ms *MemStore[T]) Update(id int64, data T) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Set the ID on the data
	data.SetID(id)

	for i, item := range ms.items {
		if item.GetID() == id {
			ms.markDirty()
			ms.items[i] = data
			return nil
		}
	}

	// Mark as dirty for background save
	ms.markDirty()
	return nil
}

// Add adds a new item with auto-generated ID
func (ms *MemStore[T]) Add(data T) (int64, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Generate new ID
	id := ms.storage.NextID
	ms.storage.NextID++

	// Set ID on the data
	data.SetID(id)

	// Add to in-memory cache
	ms.items = append(ms.items, data)

	// Mark as dirty for background save
	ms.markDirty()

	return id, nil
}

// Delete removes an item by ID
func (ms *MemStore[T]) Delete(id int64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for i, item := range ms.items {
		if item.GetID() == id {
			ms.markDirty()
			ms.items = append(ms.items[:i], ms.items[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("item with id %d not found", id)
}
