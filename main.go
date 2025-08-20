package main

import (
	"log"
	"net/http"
	"os"

	"laundry-scheduler/handlers"
	"laundry-scheduler/models"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}
	log.Printf("Starting server from directory: %s", wd)

	queue := models.NewLaundryQueue()
	webHandler := handlers.NewWebHandler(queue)

	setupRoutes(webHandler)
	setupStaticFiles()

	port := handlers.DefaultPort
	log.Printf("Server starting on http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func setupRoutes(handler *handlers.WebHandler) {
	http.HandleFunc("/", handler.Index)
	http.HandleFunc("/api/queue", handler.GetQueue)
	http.HandleFunc("/api/form", handler.GetForm)
	http.HandleFunc("/api/queue/add", handler.AddToQueue)
	http.HandleFunc("/api/queue/start/", handler.StartTimer)
	http.HandleFunc("/api/queue/", handler.RemoveFromQueue)
}

func setupStaticFiles() {
	staticDir := handlers.StaticDir
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		log.Printf("Warning: static directory not found at %s", staticDir)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
}
