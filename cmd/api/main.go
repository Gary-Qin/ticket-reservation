package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Gary-Qin/ticket-reservation/internal/events"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	// db init
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("creating pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("connecting to postgres: %v", err)
	}
	log.Println("connected to postgres")

	// handlers init
	store := events.NewMemoryStore()
	eventsHandler := events.NewHandler(store)

	// http server routes init
	mux := http.NewServeMux()
	eventsHandler.RegisterRoutes(mux)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
