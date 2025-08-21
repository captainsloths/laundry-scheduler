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

const (
	// DefaultPort is the default server port
	DefaultPort = ":8080"
	// TemplatesDir is the directory containing HTML templates
	TemplatesDir = "templates"
	// StaticDir is the directory containing static files
	StaticDir = "./static"
)

// WebHandler handles HTTP requests for the laundry queue application
type WebHandler struct {
	queue     *models.LaundryQueue
	templates *template.Template
}

// NewWebHandler creates a new web handler with initialized templates
func NewWebHandler(queue *models.LaundryQueue) *WebHandler {
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

	templatePath := filepath.Join(TemplatesDir, "*.html")

	if _, err := os.Stat(TemplatesDir); os.IsNotExist(err) {
		log.Fatal("templates directory not found! Make sure you're running from the project root directory")
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob(templatePath)
	if err != nil {
		log.Fatalf("Error parsing templates: %v", err)
	}

	return &WebHandler{
		queue:     queue,
		templates: tmpl,
	}
}

// executeTemplate executes a template with common error handling
func (h *WebHandler) executeTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	w.Header().Set("Content-Type", "text/html")
	err := h.templates.ExecuteTemplate(w, templateName, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
	}
}

// renderQueue renders the queue with positions calculated
func (h *WebHandler) renderQueue(w http.ResponseWriter, templateName string) {
	items := h.queue.GetAll()

	positions := make(map[string]int)
	waitingCount := 0
	for _, item := range items {
		if item.Status == models.StatusWaiting {
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

	h.executeTemplate(w, templateName, data)
}

// Index serves the main page
func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		HasActiveLoad bool
		Items         []*models.QueueItem
	}{
		HasActiveLoad: h.queue.HasActiveLoad(),
		Items:         h.queue.GetAll(),
	}

	h.executeTemplate(w, "index.html", data)
}

// GetQueue returns the current queue as HTML
func (h *WebHandler) GetQueue(w http.ResponseWriter, r *http.Request) {
	h.renderQueue(w, "queue.html")
}

// GetForm returns the form HTML based on queue state
func (h *WebHandler) GetForm(w http.ResponseWriter, r *http.Request) {
	hasQueueItems := h.queue.HasQueueItems()
	h.executeTemplate(w, "form.html", hasQueueItems)
}

// AddToQueue handles adding a new person to the queue
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

	if h.queue.HasQueueItems() {
		h.queue.AddToQueue(name, numLoads)
	} else {
		if durationStr != "" {
			duration, err := strconv.Atoi(durationStr)
			if err != nil || duration <= 0 {
				http.Error(w, "Invalid duration", http.StatusBadRequest)
				return
			}
			h.queue.AddAndStart(name, duration, numLoads)
		} else {
			h.queue.AddToQueue(name, numLoads)
		}
	}

	h.renderQueue(w, "queue.html")
}

// StartTimer starts the timer for a queued person
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

	h.renderQueue(w, "queue.html")
}

// RemoveFromQueue removes a person from the queue
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

	h.renderQueue(w, "queue.html")
}
