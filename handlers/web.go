package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"laundry-scheduler/models"
)

type WebHandler struct {
	store     *models.ScheduleStore
	templates *template.Template
}

func NewWebHandler(store *models.ScheduleStore) *WebHandler {
	// Parse templates at initialization
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	return &WebHandler{
		store:     store,
		templates: tmpl,
	}
}

func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	err := h.templates.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	items := h.store.GetAll()

	w.Header().Set("Content-Type", "text/html")
	err := h.templates.ExecuteTemplate(w, "schedule.html", items)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) AddSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	startTimeStr := r.FormValue("start_time")
	endTimeStr := r.FormValue("end_time")

	if title == "" || startTimeStr == "" || endTimeStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Parse times
	startTime, err := time.Parse("2006-01-02T15:04", startTimeStr)
	if err != nil {
		http.Error(w, "Invalid start time format", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse("2006-01-02T15:04", endTimeStr)
	if err != nil {
		http.Error(w, "Invalid end time format", http.StatusBadRequest)
		return
	}

	// Create and add new item
	item := &models.ScheduleItem{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Title:       title,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
		CreatedAt:   time.Now(),
	}

	h.store.Add(item)

	// Return updated schedule list
	h.GetSchedule(w, r)
}

func (h *WebHandler) RemoveSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	path := r.URL.Path
	id := path[len("/api/schedule/"):]

	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	success := h.store.Remove(id)
	if !success {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	// Return updated schedule list
	h.GetSchedule(w, r)
}
