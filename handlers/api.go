package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"laundry-scheduler/models"
)

type APIHandler struct {
	store *models.ScheduleStore
}

func NewAPIHandler(store *models.ScheduleStore) *APIHandler {
	return &APIHandler{
		store: store,
	}
}

func (h *APIHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	items := h.store.GetAll()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *APIHandler) AddSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var item models.ScheduleItem
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	item.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	item.CreatedAt = time.Now()

	h.store.Add(&item)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *APIHandler) RemoveSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	path := r.URL.Path
	id := path[len("/api/json/schedule/"):]

	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	success := h.store.Remove(id)
	if !success {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
