package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	testEvent      = "bruno-mars"
	initialTickets = 100
)

// storeFixture is what the tests need from a store: the Reserver under test,
// plus ways to inspect its state that the handler itself has no use for.
type storeFixture struct {
	reserver     Reserver
	remaining    func(t *testing.T) int
	reservations func(t *testing.T) int // nil if the store doesn't record them
}

// stores lists every implementation. Each constructor returns fresh state.
var stores = map[string]func(*testing.T) *storeFixture{
	"memory":          newMemoryFixture,
	"postgres_lock":   newPostgresLockFixture,
	"postgres_atomic": newPostgresAtomicFixture,
}

// forEachStore runs fn as a subtest against every store.
func forEachStore(t *testing.T, fn func(t *testing.T, f *storeFixture)) {
	t.Helper()
	for name, newFixture := range stores {
		t.Run(name, func(t *testing.T) {
			fn(t, newFixture(t))
		})
	}
}

func newMemoryFixture(t *testing.T) *storeFixture {
	t.Helper()
	store := NewMemoryStore()
	return &storeFixture{
		reserver: store,
		remaining: func(t *testing.T) int {
			store.mu.Lock()
			defer store.mu.Unlock()
			return store.eventsMap[testEvent]
		},
	}
}

func newPostgresLockFixture(t *testing.T) *storeFixture {
	t.Helper()
	pool := newTestPool(t)
	return newPostgresFixture(pool, NewPostgresLockStore(pool))
}

func newPostgresAtomicFixture(t *testing.T) *storeFixture {
	t.Helper()
	pool := newTestPool(t)
	return newPostgresFixture(pool, NewPostgresAtomicStore(pool))
}

// newTestPool connects to the test database and resets it. It skips the test
// when TEST_DATABASE_URL is not set.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := t.Context()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("creating pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// The reset below deletes data, so refuse to run it on any other database.
	if db := pool.Config().ConnConfig.Database; !strings.HasSuffix(db, "_test") {
		t.Fatalf("refusing to reset database %q: name must end in _test", db)
	}

	if _, err := pool.Exec(ctx, "TRUNCATE reservations"); err != nil {
		t.Fatalf("truncating reservations: %v", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE events SET remaining_tickets = $1", initialTickets); err != nil {
		t.Fatalf("resetting events: %v", err)
	}
	return pool
}

// newPostgresFixture works for any store that keeps its state in Postgres.
func newPostgresFixture(pool *pgxpool.Pool, r Reserver) *storeFixture {
	// queryInt reads without locking: it only runs while no requests are in flight.
	queryInt := func(t *testing.T, sql string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(t.Context(), sql, testEvent).Scan(&n); err != nil {
			t.Fatalf("running %q: %v", sql, err)
		}
		return n
	}
	return &storeFixture{
		reserver: r,
		remaining: func(t *testing.T) int {
			t.Helper()
			return queryInt(t, "SELECT remaining_tickets FROM events WHERE id = $1")
		},
		reservations: func(t *testing.T) int {
			t.Helper()
			return queryInt(t, "SELECT count(*) FROM reservations WHERE event_id = $1")
		},
	}
}

// newTestServer starts a real HTTP server around the given Reserver.
func newTestServer(t *testing.T, r Reserver) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	NewHandler(r).RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// reserve sends one reservation request. It returns an error instead of
// calling t.Fatal, because t.Fatal must not be called from other goroutines.
func reserve(c *http.Client, url string) (status int, id string, err error) {
	resp, err := c.Post(url, "application/json", nil)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close() // without this, connections are never reused

	if resp.StatusCode == http.StatusCreated {
		var body ReserveResponse
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return resp.StatusCode, "", fmt.Errorf("decoding body: %w", err)
		}
		id = body.ID
	}
	return resp.StatusCode, id, nil
}

// TestReserveSequential is the baseline: with one request at a time, the
// logic must be exactly right.
func TestReserveSequential(t *testing.T) {
	forEachStore(t, func(t *testing.T, f *storeFixture) {
		srv := newTestServer(t, f.reserver)
		url := srv.URL + "/events/" + testEvent + "/reserve"

		for i := 1; i <= initialTickets; i++ {
			status, _, err := reserve(srv.Client(), url)
			if err != nil {
				t.Fatalf("request %d: %v", i, err)
			}
			if status != http.StatusCreated {
				t.Fatalf("request %d: got status %d, want %d", i, status, http.StatusCreated)
			}
		}

		status, _, err := reserve(srv.Client(), url)
		if err != nil {
			t.Fatal(err)
		}
		if status != http.StatusConflict {
			t.Errorf("after selling out: got status %d, want %d", status, http.StatusConflict)
		}
	})
}

func TestReserveUnknownEvent(t *testing.T) {
	forEachStore(t, func(t *testing.T, f *storeFixture) {
		srv := newTestServer(t, f.reserver)

		status, _, err := reserve(srv.Client(), srv.URL+"/events/nope/reserve")
		if err != nil {
			t.Fatal(err)
		}
		if status != http.StatusNotFound {
			t.Errorf("got status %d, want %d", status, http.StatusNotFound)
		}
	})
}

// TestConcurrentReservers fires many requests at once and checks invariants
// that must hold no matter how requests interleave.
func TestConcurrentReservers(t *testing.T) {
	forEachStore(t, testConcurrentReservers)
}

func testConcurrentReservers(t *testing.T, f *storeFixture) {
	const requests = 1000

	srv := newTestServer(t, f.reserver)
	url := srv.URL + "/events/" + testEvent + "/reserve"

	// Opening 1,000 connections at once overflows the OS listen queue on
	// Windows (connection refused/reset). Capping connections makes extra
	// requests wait on the client side; 100 in flight is still plenty of
	// concurrency to trigger the race.
	client := &http.Client{Transport: &http.Transport{
		MaxConnsPerHost:     100,
		MaxIdleConnsPerHost: 100,
	}}
	t.Cleanup(client.CloseIdleConnections)

	var (
		mu       sync.Mutex // the test's own shared state must be safe,
		sold     int        // even if the store under test isn't
		rejected int
		failed   int
		ids      = map[string]int{} // reservation id -> times it was issued
	)

	// Every goroutine blocks on start; closing it releases them all at once,
	// so they don't trickle in while the loop is still spawning them.
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range requests {
		wg.Go(func() {
			<-start
			status, id, err := reserve(client, url)

			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				failed++
			case status == http.StatusCreated:
				sold++
				ids[id]++
			case status == http.StatusConflict:
				rejected++
			default:
				failed++
			}
		})
	}
	close(start)
	wg.Wait()

	// Close waits for in-flight handlers to finish before we read the store.
	srv.Close()
	remaining := f.remaining(t)

	t.Logf("sold=%d rejected=%d failed=%d remaining=%d", sold, rejected, failed, remaining)

	if failed > 0 {
		t.Errorf("%d requests failed (transport error or unexpected status); counts are unreliable", failed)
	}

	// Invariant 1: never sell more tickets than exist.
	if sold > initialTickets {
		t.Errorf("oversold: sold %d tickets, only %d exist", sold, initialTickets)
	}

	// Invariant 2: every ticket is accounted for. If two requests read the
	// same count and both write count-1, both sell but one decrement is lost.
	if sold+remaining != initialTickets {
		t.Errorf("lost updates: sold %d + remaining %d = %d, want %d",
			sold, remaining, sold+remaining, initialTickets)
	}

	// Invariant 3: reservation ids are unique.
	for id, n := range ids {
		if n > 1 {
			t.Errorf("reservation id %s was issued %d times", id, n)
		}
	}

	// Invariant 4: what the store recorded agrees with its inventory. This
	// catches a decrement that never got its reservation row, or vice versa.
	if f.reservations != nil {
		if rows := f.reservations(t); rows+remaining != initialTickets {
			t.Errorf("inventory mismatch: %d reservation rows + %d remaining = %d, want %d",
				rows, remaining, rows+remaining, initialTickets)
		}
	}
}
