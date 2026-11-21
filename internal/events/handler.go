package events

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type Handler struct {
	reserver Reserver
}

type ReserveResponse struct {
	Event string `json:"event"`
	ID    int    `json:"id"`
}

func NewHandler(s Reserver) *Handler {
	return &Handler{reserver: s}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /events/{id}/reserve", h.reserveTicket)
}

func (h *Handler) reserveTicket(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	res, err := h.reserver.Reserve(eventID)
	if err != nil {
		switch {
		case errors.Is(err, ErrEventNotFound):
			http.Error(w, "event not found", http.StatusNotFound)
		case errors.Is(err, ErrNoMoreTickets):
			http.Error(w, "event sold out", http.StatusConflict)
		default:
			log.Println("error reserving ticket:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(ReserveResponse{Event: res.Event, ID: res.ID}); err != nil {
		log.Println("error encoding response:", err)
	}
}
