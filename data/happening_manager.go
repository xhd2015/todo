package data

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

// HappeningManager manages happenings with internal caching
type HappeningManager struct {
	service storage.HappeningService

	// Internal cache state
	mutex            sync.RWMutex
	cachedHappenings []*models.Happening
	loaded           bool
	loading          bool
}

// NewHappeningManager creates a new HappeningManager with the given service
func NewHappeningManager(service storage.HappeningService) *HappeningManager {
	return &HappeningManager{
		service: service,
	}
}

// LoadHappenings returns cached happenings immediately after first load, with async refresh
func (hm *HappeningManager) LoadHappenings(ctx context.Context) ([]*models.Happening, error) {
	hm.mutex.RLock()

	// If we have cached data, return it immediately and refresh in background
	if hm.loaded {
		cached := make([]*models.Happening, len(hm.cachedHappenings))
		copy(cached, hm.cachedHappenings)
		hm.mutex.RUnlock()

		// Start async refresh in background
		go hm.refreshAsync()

		return cached, nil
	}

	// If currently loading, wait for it to complete
	if hm.loading {
		hm.mutex.RUnlock()
		// Wait and retry (simple polling approach)
		for {
			time.Sleep(10 * time.Millisecond)
			hm.mutex.RLock()
			if hm.loaded {
				cached := make([]*models.Happening, len(hm.cachedHappenings))
				copy(cached, hm.cachedHappenings)
				hm.mutex.RUnlock()
				return cached, nil
			}
			if !hm.loading {
				// Loading failed, break and try loading ourselves
				hm.mutex.RUnlock()
				break
			}
			hm.mutex.RUnlock()
		}
		hm.mutex.RLock()
	}

	hm.mutex.RUnlock()

	// First time loading - do it synchronously
	return hm.loadAndCache(ctx)
}

// AddHappening adds a new happening and updates the cache internally
func (hm *HappeningManager) AddHappening(ctx context.Context, content string) (*models.Happening, error) {
	happening := &models.Happening{
		Content: content,
	}

	newHappening, err := hm.service.Add(ctx, happening)
	if err != nil {
		return nil, err
	}

	// Update cache internally
	hm.addToCache(newHappening)

	return newHappening, nil
}

// UpdateHappening updates a happening and updates the cache internally
func (hm *HappeningManager) UpdateHappening(ctx context.Context, id int64, update *models.HappeningOptional) (*models.Happening, error) {
	updatedHappening, err := hm.service.Update(ctx, id, update)
	if err != nil {
		return nil, err
	}

	// Update cache internally
	hm.updateInCache(updatedHappening)

	return updatedHappening, nil
}

// DeleteHappening deletes a happening and updates the cache internally
func (hm *HappeningManager) DeleteHappening(ctx context.Context, id int64) error {
	err := hm.service.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Remove from cache internally
	hm.removeFromCache(id)

	return nil
}

// InvalidateCache clears the cache, forcing next load to be fresh
func (hm *HappeningManager) InvalidateCache() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.cachedHappenings = nil
	hm.loaded = false
	hm.loading = false
}

// loadAndCache performs the actual loading and caching
func (hm *HappeningManager) loadAndCache(ctx context.Context) ([]*models.Happening, error) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	// Double-check in case another goroutine loaded while we were waiting for the lock
	if hm.loaded {
		cached := make([]*models.Happening, len(hm.cachedHappenings))
		copy(cached, hm.cachedHappenings)
		return cached, nil
	}

	// Mark as loading
	hm.loading = true
	defer func() {
		hm.loading = false
	}()

	// Load happenings from storage
	happenings, _, err := hm.service.List(storage.HappeningListOptions{
		Limit: 20,
	})
	if err != nil {
		return nil, err
	}

	// Sort by create time ASC
	sort.Slice(happenings, func(i, j int) bool {
		return happenings[i].CreateTime.Before(happenings[j].CreateTime)
	})

	// Cache the results
	hm.cachedHappenings = happenings
	hm.loaded = true

	// Return a copy to avoid external modifications
	cached := make([]*models.Happening, len(happenings))
	copy(cached, happenings)
	return cached, nil
}

// refreshAsync refreshes the cache in the background
func (hm *HappeningManager) refreshAsync() {
	// Load fresh data
	happenings, _, err := hm.service.List(storage.HappeningListOptions{
		Limit: 20,
	})
	if err != nil {
		// If refresh fails, keep the old cache
		return
	}

	// Sort by create time ASC
	sort.Slice(happenings, func(i, j int) bool {
		return happenings[i].CreateTime.Before(happenings[j].CreateTime)
	})

	// Update cache
	hm.mutex.Lock()
	hm.cachedHappenings = happenings
	hm.mutex.Unlock()
}

// addToCache adds a new happening to the cache
func (hm *HappeningManager) addToCache(happening *models.Happening) {
	if happening == nil {
		return
	}

	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if !hm.loaded {
		return // No cache to update
	}

	// Add to cache and re-sort
	hm.cachedHappenings = append(hm.cachedHappenings, happening)
	sort.Slice(hm.cachedHappenings, func(i, j int) bool {
		return hm.cachedHappenings[i].CreateTime.Before(hm.cachedHappenings[j].CreateTime)
	})
}

// updateInCache updates a happening in the cache
func (hm *HappeningManager) updateInCache(happening *models.Happening) {
	if happening == nil {
		return
	}

	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if !hm.loaded {
		return // No cache to update
	}

	// Find and update the happening in cache
	for i, cached := range hm.cachedHappenings {
		if cached.ID == happening.ID {
			hm.cachedHappenings[i] = happening
			// Re-sort in case the update time changed the order
			sort.Slice(hm.cachedHappenings, func(i, j int) bool {
				return hm.cachedHappenings[i].CreateTime.Before(hm.cachedHappenings[j].CreateTime)
			})
			break
		}
	}
}

// removeFromCache removes a happening from the cache
func (hm *HappeningManager) removeFromCache(id int64) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if !hm.loaded {
		return // No cache to update
	}

	// Find and remove the happening from cache
	for i, cached := range hm.cachedHappenings {
		if cached.ID == id {
			hm.cachedHappenings = append(hm.cachedHappenings[:i], hm.cachedHappenings[i+1:]...)
			break
		}
	}
}
