package events

var _ Reserver = (*MemoryStore)(nil)

type MemoryStore struct {
	eventsMap map[string]int
	nextID    int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		eventsMap: initializeEvents(),
	}
}

func (s *MemoryStore) Reserve(eventID string) (Reservation, error) {
	ticketsLeft, ok := s.eventsMap[eventID]
	if !ok {
		return Reservation{}, ErrEventNotFound
	}

	if ticketsLeft <= 0 {
		return Reservation{}, ErrNoMoreTickets
	}

	s.eventsMap[eventID]--
	s.nextID++
	return Reservation{
		Event: eventID,
		ID:    s.nextID,
	}, nil
}

func initializeEvents() map[string]int {
	events := make(map[string]int)

	events["ariana-grande"] = 100
	events["bruno-mars"] = 100
	events["calvin-harris"] = 100

	return events
}
