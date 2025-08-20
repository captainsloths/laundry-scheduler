package main

import (
	"log"
	"net/http"
	"os"

	"laundry-scheduler/handlers"
	"laundry-scheduler/models"
)

func main() {
	wd, _ := os.Getwd()
	log.Printf("Starting server from directory: %s", wd)

	// Initialize the laundry queue
	queue := models.NewLaundryQueue()

	// Create handlers
	webHandler := handlers.NewWebHandler(queue)

	// Routes
	http.HandleFunc("/", webHandler.Index)
	http.HandleFunc("/api/queue", webHandler.GetQueue)
	http.HandleFunc("/api/form", webHandler.GetForm)
	http.HandleFunc("/api/queue/add", webHandler.AddToQueue)
	http.HandleFunc("/api/queue/start/", webHandler.StartTimer)
	http.HandleFunc("/api/queue/", webHandler.RemoveFromQueue)

	// Serve static files
	staticDir := "./static"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		log.Printf("Warning: static directory not found at %s", staticDir)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	port := ":8080"
	log.Printf("Server starting on http://localhost%s", port)
	log.Printf("Make sure you have the following structure:")
	log.Printf("  - templates/index.html")
	log.Printf("  - templates/form.html")
	log.Printf("  - templates/queue.html")
	log.Printf("  - static/style.css")
	log.Fatal(http.ListenAndServe(port, nil))
}
