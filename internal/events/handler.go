package events

import (
	"errors"
	"log"
	"net/http"
)

type Handler struct {
	service Reserver
}

func NewHandler(s Reserver) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /events/{id}/reserve", h.reserveTicket)
}

func (h *Handler) reserveTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	log.Printf("attempting to reserve ticket with id %s", id)

	err := h.service.Reserve(id)
	if err != nil {
		log.Println("error booking ticket:", err)
		switch {
		case errors.Is(err, ErrEventNotFound):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, ErrNoMoreTickets):
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}

	}
}
