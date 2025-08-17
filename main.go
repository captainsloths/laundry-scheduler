package main

import (
	"log"
	"net/http"

	"laundry-scheduler/handlers"
	"laundry-scheduler/models"
)

func main() {
	// Initialize the store
	store := models.NewScheduleStore()

	// Create handlers with store
	webHandler := handlers.NewWebHandler(store)
	apiHandler := handlers.NewAPIHandler(store)

	// Routes
	http.HandleFunc("/", webHandler.Index)
	http.HandleFunc("/api/schedule", webHandler.GetSchedule)
	http.HandleFunc("/api/schedule/add", webHandler.AddSchedule)
	http.HandleFunc("/api/schedule/", webHandler.RemoveSchedule)

	// JSON API endpoints
	http.HandleFunc("/api/json/schedule", apiHandler.GetSchedule)
	http.HandleFunc("/api/json/schedule/add", apiHandler.AddSchedule)

	// Serve static files (optional)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	port := ":8080"
	log.Printf("Server starting on http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
