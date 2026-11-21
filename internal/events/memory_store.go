package events

import (
	"errors"
)

var (
	ErrEventNotFound = errors.New("event id does not exist")
	ErrNoMoreTickets = errors.New("no tickets left")
)

type MemoryStore struct {
	eventsMap map[string]int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		eventsMap: initializeEvents(),
	}
}

func (s *MemoryStore) Reserve(id string) error {
	ticketsLeft, ok := s.eventsMap[id]
	if !ok {
		return ErrEventNotFound
	}

	if ticketsLeft <= 0 {
		return ErrNoMoreTickets
	}

	s.eventsMap[id]--
	return nil
}

func initializeEvents() map[string]int {
	events := make(map[string]int)

	events["ariana-grande"] = 100
	events["bruno-mars"] = 100
	events["calvin-harris"] = 100

	return events
}
