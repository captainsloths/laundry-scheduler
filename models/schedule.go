package models

import (
	"sync"
	"time"
)

// ScheduleItem represents a single item in the schedule
type ScheduleItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	CreatedAt   time.Time `json:"created_at"`
}

// ScheduleStore manages the schedule items in memory
type ScheduleStore struct {
	mu    sync.RWMutex
	items map[string]*ScheduleItem
}

// NewScheduleStore creates a new schedule store
func NewScheduleStore() *ScheduleStore {
	return &ScheduleStore{
		items: make(map[string]*ScheduleItem),
	}
}

// Add adds a new item to the schedule
func (s *ScheduleStore) Add(item *ScheduleItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.ID] = item
}

// Remove removes an item from the schedule
func (s *ScheduleStore) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[id]; exists {
		delete(s.items, id)
		return true
	}
	return false
}

// GetAll returns all schedule items sorted by start time
func (s *ScheduleStore) GetAll() []*ScheduleItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*ScheduleItem, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}

	// Sort by start time
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].StartTime.After(items[j].StartTime) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	return items
}

// Get returns a single schedule item by ID
func (s *ScheduleStore) Get(id string) (*ScheduleItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, exists := s.items[id]
	return item, exists
}
