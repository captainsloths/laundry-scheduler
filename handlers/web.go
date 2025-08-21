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
		"formatTimeRange": func(minutes int, suffix string) string {
			if minutes <= 0 {
				return "Complete"
			}
			if minutes < 60 {
				return fmt.Sprintf("%d min%s", minutes, suffix)
			}
			hours := minutes / 60
			mins := minutes % 60
			if mins > 0 {
				return fmt.Sprintf("%dh %dm%s", hours, mins, suffix)
			}
			return fmt.Sprintf("%dh%s", hours, suffix)
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
	if err := h.templates.ExecuteTemplate(w, templateName, data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// renderQueue renders the queue with positions calculated
func (h *WebHandler) renderQueue(w http.ResponseWriter, templateName string) {
	items := h.queue.GetAll()
	positions := make(map[string]int)

	for _, item := range items {
		if item.Status == models.StatusWaiting {
			positions[item.ID] = len(positions) + 1
		}
	}

	h.executeTemplate(w, templateName, struct {
		Items     []*models.QueueItem
		Positions map[string]int
	}{items, positions})
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

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	numLoads, err := strconv.Atoi(r.FormValue("num_loads"))
	if err != nil || numLoads <= 0 || numLoads > 10 {
		http.Error(w, "Invalid number of loads (must be 1-10)", http.StatusBadRequest)
		return
	}

	if h.queue.HasQueueItems() {
		h.queue.AddToQueue(name, numLoads)
	} else if durationStr := r.FormValue("duration"); durationStr != "" {
		if duration, err := strconv.Atoi(durationStr); err == nil && duration > 0 {
			h.queue.AddAndStart(name, duration, numLoads)
		} else {
			http.Error(w, "Invalid duration", http.StatusBadRequest)
			return
		}
	} else {
		h.queue.AddToQueue(name, numLoads)
	}

	h.renderQueue(w, "queue.html")
}

// StartTimer starts the timer for a queued person
func (h *WebHandler) StartTimer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[len("/api/queue/start/"):]
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	duration, err := strconv.Atoi(r.FormValue("duration"))
	if err != nil || duration <= 0 {
		http.Error(w, "Invalid duration", http.StatusBadRequest)
		return
	}

	if !h.queue.StartTimer(id, duration) {
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

	id := r.URL.Path[len("/api/queue/"):]
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if !h.queue.Remove(id) {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	h.renderQueue(w, "queue.html")
}
