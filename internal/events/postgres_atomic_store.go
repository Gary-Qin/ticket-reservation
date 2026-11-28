package events

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ Reserver = (*PostgresAtomicStore)(nil)

type PostgresAtomicStore struct {
	pool *pgxpool.Pool
}

func NewPostgresAtomicStore(pool *pgxpool.Pool) *PostgresAtomicStore {
	return &PostgresAtomicStore{pool: pool}
}

func (s *PostgresAtomicStore) Reserve(ctx context.Context, eventID string) (Reservation, error) {
	uid, err := uuid.NewV7()
	if err != nil {
		return Reservation{}, fmt.Errorf("generating uuid: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reservation{}, fmt.Errorf("starting psql transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var remainingTickets int

	query := `
		UPDATE events
		SET remaining_tickets = remaining_tickets - 1
		WHERE id = $1
		AND remaining_tickets > 0
		RETURNING remaining_tickets
	`
	if err := tx.QueryRow(ctx, query, eventID).Scan(&remainingTickets); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var eventExists bool
			if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM events WHERE id = $1)", eventID).Scan(&eventExists); err != nil {
				return Reservation{}, fmt.Errorf("checking if event exists: %w", err)
			}

			if eventExists {
				return Reservation{}, ErrNoMoreTickets
			}
			return Reservation{}, ErrEventNotFound
		}
		return Reservation{}, fmt.Errorf("querying & updating remaining tickets: %w", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO reservations (id, event_id) VALUES ($1, $2)", uid, eventID)
	if err != nil {
		return Reservation{}, fmt.Errorf("inserting reservations: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Reservation{}, fmt.Errorf("committing transaction: %w", err)
	}

	return Reservation{
		Event: eventID,
		ID:    uid,
	}, nil
}
