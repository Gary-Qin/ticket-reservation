package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

var _ Reserver = (*MemoryStore)(nil)

type MemoryStore struct {
	mu        sync.Mutex
	eventsMap map[string]int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		eventsMap: initializeEvents(),
	}
}

func (s *MemoryStore) Reserve(ctx context.Context, eventID string) (Reservation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ticketsLeft, ok := s.eventsMap[eventID]
	if !ok {
		return Reservation{}, ErrEventNotFound
	}
	if ticketsLeft <= 0 {
		return Reservation{}, ErrNoMoreTickets
	}

	uid, err := uuid.NewV7()
	if err != nil {
		return Reservation{}, fmt.Errorf("generating uuid: %w", err)
	}

	s.eventsMap[eventID]--

	return Reservation{
		Event: eventID,
		ID:    uid,
	}, nil
}

func initializeEvents() map[string]int {
	events := make(map[string]int)

	events["ariana-grande"] = 100
	events["bruno-mars"] = 100
	events["calvin-harris"] = 100

	return events
}
