package main

import (
	"log"
	"net/http"

	"github.com/Gary-Qin/ticket-reservation/internal/events"
)

func main() {
	store := events.NewMemoryStore()
	eventsHandler := events.NewHandler(store)

	mux := http.NewServeMux()
	eventsHandler.RegisterRoutes(mux)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
