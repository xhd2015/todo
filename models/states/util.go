package states

import "sync"

// MutexMap provides thread-safe access to a map[int64]bool
type MutexMap struct {
	mu   sync.RWMutex
	data map[int64]bool
}

// NewMutexMap creates a new thread-safe map
func NewMutexMap() *MutexMap {
	data := make(map[int64]bool)
	data[GROUP_OTHER_ID] = true
	return &MutexMap{
		data: data,
	}
}

// Get safely reads a value from the map
func (m *MutexMap) Get(key int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[key]
}

// Set safely writes a value to the map
func (m *MutexMap) Set(key int64, value bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

// Toggle safely toggles a value in the map
func (m *MutexMap) Toggle(key int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = !m.data[key]
}

// Copy safely returns a copy of the entire map
func (m *MutexMap) Copy() map[int64]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[int64]bool)
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

// IsEmpty safely checks if the map is empty
func (m *MutexMap) IsEmpty() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data) == 0
}
