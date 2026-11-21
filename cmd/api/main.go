package main

import (
	"log"
	"net/http"

	"github.com/Gary-Qin/ticket-reservation/internal/events"
)

func main() {
	eventsService := events.NewMemoryStore()
	eventsHandler := events.NewHandler(eventsService)

	mux := http.NewServeMux()
	eventsHandler.RegisterRoutes(mux)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
