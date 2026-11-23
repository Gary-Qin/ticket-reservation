package events

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ Reserver = (*PostgresLockStore)(nil)

type PostgresLockStore struct {
	pool *pgxpool.Pool
}

func NewPostgresLockStore(pool *pgxpool.Pool) *PostgresLockStore {
	return &PostgresLockStore{pool: pool}
}

func (s *PostgresLockStore) Reserve(ctx context.Context, eventID string) (Reservation, error) {
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
	if err := tx.QueryRow(ctx, "SELECT remaining_tickets FROM events WHERE id = $1 FOR UPDATE",
		eventID).Scan(&remainingTickets); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Reservation{}, ErrEventNotFound
		}
		return Reservation{}, fmt.Errorf("querying remaining tickets: %w", err)
	}

	if remainingTickets <= 0 {
		return Reservation{}, ErrNoMoreTickets
	}

	remainingTickets--
	_, err = tx.Exec(ctx, "UPDATE events SET remaining_tickets = $1 WHERE id = $2", remainingTickets, eventID)
	if err != nil {
		return Reservation{}, fmt.Errorf("updating remaining tickets: %w", err)
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
