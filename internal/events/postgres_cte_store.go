package events

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ Reserver = (*PostgresCTEStore)(nil)

type PostgresCTEStore struct {
	pool *pgxpool.Pool
}

func NewPostgresCTEStore(pool *pgxpool.Pool) *PostgresCTEStore {
	return &PostgresCTEStore{pool: pool}
}

func (s *PostgresCTEStore) Reserve(ctx context.Context, eventID string) (Reservation, error) {
	uid, err := uuid.NewV7()
	if err != nil {
		return Reservation{}, fmt.Errorf("generating uuid: %w", err)
	}

	query := `
		WITH updated AS (
			UPDATE events
			SET remaining_tickets = remaining_tickets - 1
			WHERE id = $1
			AND remaining_tickets > 0
			RETURNING id
		)
		INSERT INTO reservations (id, event_id)
		SELECT $2, id FROM updated
	`

	tag, err := s.pool.Exec(ctx, query, eventID, uid)
	if err != nil {
		return Reservation{}, fmt.Errorf("reserving ticket: %w", err)
	}

	if tag.RowsAffected() == 0 {
		var eventExists bool
		if err := s.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM events WHERE id = $1)", eventID).Scan(&eventExists); err != nil {
			return Reservation{}, fmt.Errorf("checking if event exists: %w", err)
		}

		if eventExists {
			return Reservation{}, ErrNoMoreTickets
		}
		return Reservation{}, ErrEventNotFound
	}

	return Reservation{
		Event: eventID,
		ID:    uid,
	}, nil
}
