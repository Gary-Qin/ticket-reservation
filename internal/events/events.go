// Package events handles ticket reservation for events
package events

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrEventNotFound = errors.New("event id does not exist")
	ErrNoMoreTickets = errors.New("no tickets left")
)

type Reserver interface {
	Reserve(ctx context.Context, eventID string) (Reservation, error)
}

type Reservation struct {
	Event string
	ID    uuid.UUID
}
