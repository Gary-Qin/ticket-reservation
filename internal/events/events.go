// Package events handles ticket reservation for events
package events

import "errors"

var (
	ErrEventNotFound = errors.New("event id does not exist")
	ErrNoMoreTickets = errors.New("no tickets left")
)

type Reserver interface {
	Reserve(eventID string) (Reservation, error)
}

type Reservation struct {
	Event string
	ID    int
}
