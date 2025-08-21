package models

import (
	"sync"
	"time"
)

const (
	// StatusWaiting indicates a queue item is waiting to start
	StatusWaiting = "waiting"
	// StatusInProgress indicates a queue item is currently running
	StatusInProgress = "in_progress"
	// StatusCompleted indicates a queue item has finished
	StatusCompleted = "completed"

	// AutoRemoveDelay is how long completed items stay before auto-removal
	AutoRemoveDelay = 5 * time.Minute
	// BackgroundWorkerInterval is how often the background worker runs
	BackgroundWorkerInterval = 30 * time.Second
)

// QueueItem represents a person in the laundry queue
type QueueItem struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	Duration    int        `json:"duration,omitempty"`
	NumLoads    int        `json:"num_loads"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	QueuedAt    time.Time  `json:"queued_at"`
}

// GetRemainingMinutes returns how many minutes are left
func (q *QueueItem) GetRemainingMinutes() int {
	if q.Status != StatusInProgress || q.StartTime == nil || q.Duration == 0 {
		return 0
	}
	endTime := q.StartTime.Add(time.Duration(q.Duration) * time.Minute)
	remaining := time.Until(endTime).Minutes()
	if remaining < 0 {
		return 0
	}
	return int(remaining)
}

// IsTimerExpired checks if the timer has expired
func (q *QueueItem) IsTimerExpired() bool {
	return q.GetRemainingMinutes() <= 0
}

// ShouldAutoRemove checks if completed item should be removed
func (q *QueueItem) ShouldAutoRemove() bool {
	if q.Status != StatusCompleted || q.CompletedAt == nil {
		return false
	}
	return time.Since(*q.CompletedAt) > AutoRemoveDelay
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
	go queue.backgroundWorker()
	return queue
}

func (q *LaundryQueue) backgroundWorker() {
	ticker := time.NewTicker(BackgroundWorkerInterval)
	defer ticker.Stop()

	for range ticker.C {
		q.mu.Lock()
		newItems := make([]*QueueItem, 0)
		for _, item := range q.items {
			if item.Status == StatusInProgress && item.IsTimerExpired() {
				item.Status = StatusCompleted
				now := time.Now()
				item.CompletedAt = &now
			}

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
		Status:   StatusWaiting,
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
		if item.ID == id && item.Status == StatusWaiting {
			now := time.Now()
			item.StartTime = &now
			item.Duration = duration
			item.Status = StatusInProgress
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
		Status:    StatusInProgress,
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
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]*QueueItem, len(q.items))
	copy(result, q.items)
	return result
}

// HasActiveLoad checks if anyone has a load currently running
func (q *LaundryQueue) HasActiveLoad() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, item := range q.items {
		if item.Status == StatusInProgress && !item.IsTimerExpired() {
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
		if item.Status == StatusWaiting || (item.Status == StatusInProgress && !item.IsTimerExpired()) {
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
		if item.Status == StatusWaiting {
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
