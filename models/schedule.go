package models

import (
	"sync"
	"time"
)

// QueueItem represents a person in the laundry queue
type QueueItem struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"` // "waiting", "in_progress", "completed"
	StartTime   *time.Time `json:"start_time,omitempty"`
	Duration    int        `json:"duration,omitempty"` // Duration in minutes
	NumLoads    int        `json:"num_loads"`          // Number of loads to wash
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	QueuedAt    time.Time  `json:"queued_at"`
}

// GetEndTime calculates when the laundry will be done
func (q *QueueItem) GetEndTime() *time.Time {
	if q.StartTime == nil || q.Duration == 0 {
		return nil
	}
	endTime := q.StartTime.Add(time.Duration(q.Duration) * time.Minute)
	return &endTime
}

// IsTimerExpired checks if the timer has expired
func (q *QueueItem) IsTimerExpired() bool {
	if q.Status != "in_progress" || q.StartTime == nil {
		return false
	}
	endTime := q.GetEndTime()
	return endTime != nil && time.Now().After(*endTime)
}

// GetRemainingMinutes returns how many minutes are left
func (q *QueueItem) GetRemainingMinutes() int {
	if q.Status != "in_progress" || q.StartTime == nil {
		return 0
	}
	endTime := q.GetEndTime()
	if endTime == nil {
		return 0
	}
	remaining := time.Until(*endTime).Minutes()
	if remaining < 0 {
		return 0
	}
	return int(remaining)
}

// ShouldAutoRemove checks if completed item should be removed (after 5 minutes)
func (q *QueueItem) ShouldAutoRemove() bool {
	if q.Status != "completed" || q.CompletedAt == nil {
		return false
	}
	return time.Since(*q.CompletedAt) > 5*time.Minute
}

// LaundryQueue manages the queue
type LaundryQueue struct {
	mu    sync.RWMutex
	items []*QueueItem
}

// NewLaundryQueue creates a new queue
func NewLaundryQueue() *LaundryQueue {
	queue := &LaundryQueue{
		items: make([]*QueueItem, 0),
	}

	// Start background worker for status updates
	go queue.backgroundWorker()

	return queue
}

// backgroundWorker updates statuses and removes old completed items
func (q *LaundryQueue) backgroundWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		q.mu.Lock()

		// Update statuses and remove old completed items
		newItems := make([]*QueueItem, 0)
		for _, item := range q.items {
			// Check if timer expired
			if item.Status == "in_progress" && item.IsTimerExpired() {
				item.Status = "completed"
				now := time.Now()
				item.CompletedAt = &now
			}

			// Keep item if it shouldn't be auto-removed
			if !item.ShouldAutoRemove() {
				newItems = append(newItems, item)
			}
		}
		q.items = newItems

		q.mu.Unlock()
	}
}

// AddToQueue adds a new person to the queue
func (q *LaundryQueue) AddToQueue(name string, numLoads int) *QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()

	item := &QueueItem{
		ID:       time.Now().Format("20060102150405") + "-" + name,
		Name:     name,
		Status:   "waiting",
		NumLoads: numLoads,
		QueuedAt: time.Now(),
	}

	q.items = append(q.items, item)
	return item
}

// StartTimer starts the timer for a queued person
func (q *LaundryQueue) StartTimer(id string, duration int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, item := range q.items {
		if item.ID == id && item.Status == "waiting" {
			now := time.Now()
			item.StartTime = &now
			item.Duration = duration
			item.Status = "in_progress"
			return true
		}
	}
	return false
}

// AddAndStart adds a new person and immediately starts their timer
func (q *LaundryQueue) AddAndStart(name string, duration int, numLoads int) *QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	item := &QueueItem{
		ID:        time.Now().Format("20060102150405") + "-" + name,
		Name:      name,
		Status:    "in_progress",
		StartTime: &now,
		Duration:  duration,
		NumLoads:  numLoads,
		QueuedAt:  now,
	}

	q.items = append(q.items, item)
	return item
}

// GetAll returns all queue items
func (q *LaundryQueue) GetAll() []*QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Update statuses before returning
	for _, item := range q.items {
		if item.Status == "in_progress" && item.IsTimerExpired() {
			item.Status = "completed"
			if item.CompletedAt == nil {
				now := time.Now()
				item.CompletedAt = &now
			}
		}
	}

	// Remove items that should be auto-removed
	newItems := make([]*QueueItem, 0)
	for _, item := range q.items {
		if !item.ShouldAutoRemove() {
			newItems = append(newItems, item)
		}
	}
	q.items = newItems

	// Return a copy
	result := make([]*QueueItem, len(q.items))
	copy(result, q.items)
	return result
}

// HasActiveLoad checks if anyone has a load currently running
func (q *LaundryQueue) HasActiveLoad() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, item := range q.items {
		if item.Status == "in_progress" && !item.IsTimerExpired() {
			return true
		}
	}
	return false
}

// HasQueueItems checks if there are any items in the queue (waiting or in progress)
func (q *LaundryQueue) HasQueueItems() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, item := range q.items {
		if item.Status == "waiting" || (item.Status == "in_progress" && !item.IsTimerExpired()) {
			return true
		}
	}
	return false
}

// GetQueuePosition returns the position of a person in the waiting queue
func (q *LaundryQueue) GetQueuePosition(id string) int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	position := 0
	for _, item := range q.items {
		if item.Status == "waiting" {
			position++
			if item.ID == id {
				return position
			}
		}
	}
	return -1
}

// Remove removes an item from the queue
func (q *LaundryQueue) Remove(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, item := range q.items {
		if item.ID == id {
			q.items = append(q.items[:i], q.items[i+1:]...)
			return true
		}
	}
	return false
}
