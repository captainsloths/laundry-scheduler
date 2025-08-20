package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"laundry-scheduler/models"
)

type WebHandler struct {
	queue     *models.LaundryQueue
	templates *template.Template
}

func NewWebHandler(queue *models.LaundryQueue) *WebHandler {
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory: %v", err)
	}
	log.Printf("Working directory: %s", wd)

	// Custom template functions
	funcMap := template.FuncMap{
		"formatTime": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Format("3:04 PM")
		},
		"formatDuration": func(minutes int) string {
			if minutes < 60 {
				return fmt.Sprintf("%d min", minutes)
			}
			hours := minutes / 60
			mins := minutes % 60
			if mins > 0 {
				return fmt.Sprintf("%dh %dm", hours, mins)
			}
			return fmt.Sprintf("%dh", hours)
		},
		"getRemainingTime": func(item *models.QueueItem) string {
			minutes := item.GetRemainingMinutes()
			if minutes <= 0 {
				return "Complete"
			} else if minutes < 60 {
				return fmt.Sprintf("%d min remaining", minutes)
			} else {
				hours := minutes / 60
				mins := minutes % 60
				if mins > 0 {
					return fmt.Sprintf("%dh %dm remaining", hours, mins)
				}
				return fmt.Sprintf("%dh remaining", hours)
			}
		},
	}

	templatePath := filepath.Join("templates", "*.html")
	log.Printf("Looking for templates at: %s", templatePath)

	if _, err := os.Stat("templates"); os.IsNotExist(err) {
		log.Fatal("templates directory not found! Make sure you're running from the project root directory")
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob(templatePath)
	if err != nil {
		log.Fatalf("Error parsing templates: %v", err)
	}

	log.Printf("Templates loaded successfully")

	return &WebHandler{
		queue:     queue,
		templates: tmpl,
	}
}

func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		HasActiveLoad bool
		Items         []*models.QueueItem
	}{
		HasActiveLoad: h.queue.HasActiveLoad(),
		Items:         h.queue.GetAll(),
	}

	w.Header().Set("Content-Type", "text/html")
	err := h.templates.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
	}
}

func (h *WebHandler) GetQueue(w http.ResponseWriter, r *http.Request) {
	items := h.queue.GetAll()

	// Calculate positions for all waiting items
	positions := make(map[string]int)
	waitingCount := 0
	for _, item := range items {
		if item.Status == "waiting" {
			waitingCount++
			positions[item.ID] = waitingCount
		}
	}

	// Create a struct with items and positions map
	data := struct {
		Items     []*models.QueueItem
		Positions map[string]int
	}{
		Items:     items,
		Positions: positions,
	}

	w.Header().Set("Content-Type", "text/html")
	err := h.templates.ExecuteTemplate(w, "queue.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
	}
}

func (h *WebHandler) GetForm(w http.ResponseWriter, r *http.Request) {
	hasQueueItems := h.queue.HasQueueItems()

	w.Header().Set("Content-Type", "text/html")
	err := h.templates.ExecuteTemplate(w, "form.html", hasQueueItems)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
	}
}

func (h *WebHandler) AddToQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	durationStr := r.FormValue("duration")
	numLoadsStr := r.FormValue("num_loads")

	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	if numLoadsStr == "" {
		http.Error(w, "Number of loads is required", http.StatusBadRequest)
		return
	}

	numLoads, err := strconv.Atoi(numLoadsStr)
	if err != nil || numLoads <= 0 || numLoads > 10 {
		http.Error(w, "Invalid number of loads (must be 1-10)", http.StatusBadRequest)
		return
	}

	// Check if there are any items in the queue
	if h.queue.HasQueueItems() {
		// Just add to queue without starting
		h.queue.AddToQueue(name, numLoads)
	} else {
		// No items in queue, start immediately if duration provided
		if durationStr != "" {
			duration, err := strconv.Atoi(durationStr)
			if err != nil || duration <= 0 {
				http.Error(w, "Invalid duration", http.StatusBadRequest)
				return
			}
			h.queue.AddAndStart(name, duration, numLoads)
		} else {
			// Just queue if no duration
			h.queue.AddToQueue(name, numLoads)
		}
	}

	// Return updated queue only
	w.Header().Set("Content-Type", "text/html")

	items := h.queue.GetAll()

	// Calculate positions
	positions := make(map[string]int)
	waitingCount := 0
	for _, item := range items {
		if item.Status == "waiting" {
			waitingCount++
			positions[item.ID] = waitingCount
		}
	}

	data := struct {
		Items     []*models.QueueItem
		Positions map[string]int
	}{
		Items:     items,
		Positions: positions,
	}
	h.templates.ExecuteTemplate(w, "queue.html", data)
}

func (h *WebHandler) StartTimer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	id := path[len("/api/queue/start/"):]

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	durationStr := r.FormValue("duration")
	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration <= 0 {
		http.Error(w, "Invalid duration", http.StatusBadRequest)
		return
	}

	success := h.queue.StartTimer(id, duration)
	if !success {
		http.Error(w, "Could not start timer", http.StatusBadRequest)
		return
	}

	// Return updated queue
	items := h.queue.GetAll()

	// Calculate positions
	positions := make(map[string]int)
	waitingCount := 0
	for _, item := range items {
		if item.Status == "waiting" {
			waitingCount++
			positions[item.ID] = waitingCount
		}
	}

	data := struct {
		Items     []*models.QueueItem
		Positions map[string]int
	}{
		Items:     items,
		Positions: positions,
	}

	w.Header().Set("Content-Type", "text/html")
	err = h.templates.ExecuteTemplate(w, "queue.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
	}
}

func (h *WebHandler) RemoveFromQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	id := path[len("/api/queue/"):]

	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	success := h.queue.Remove(id)
	if !success {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	// Return updated queue
	items := h.queue.GetAll()

	// Calculate positions
	positions := make(map[string]int)
	waitingCount := 0
	for _, item := range items {
		if item.Status == "waiting" {
			waitingCount++
			positions[item.ID] = waitingCount
		}
	}

	data := struct {
		Items     []*models.QueueItem
		Positions map[string]int
	}{
		Items:     items,
		Positions: positions,
	}

	w.Header().Set("Content-Type", "text/html")
	err := h.templates.ExecuteTemplate(w, "queue.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
	}
}
