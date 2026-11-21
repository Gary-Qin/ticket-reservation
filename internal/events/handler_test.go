package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const testEvent = "bruno-mars"

// newTestServer starts a real HTTP server around a fresh MemoryStore.
// It returns the store too, so tests can inspect its state afterwards.
func newTestServer(t *testing.T) (*httptest.Server, *MemoryStore) {
	t.Helper()
	store := NewMemoryStore()
	mux := http.NewServeMux()
	NewHandler(store).RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, store
}

// reserve sends one reservation request. It returns an error instead of
// calling t.Fatal, because t.Fatal must not be called from other goroutines.
func reserve(c *http.Client, url string) (status int, id int, err error) {
	resp, err := c.Post(url, "application/json", nil)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close() // without this, connections are never reused

	if resp.StatusCode == http.StatusCreated {
		var body ReserveResponse
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return resp.StatusCode, 0, fmt.Errorf("decoding body: %w", err)
		}
		id = body.ID
	}
	return resp.StatusCode, id, nil
}

// TestReserveSequential is the baseline: with one request at a time, the
// logic must be exactly right.
func TestReserveSequential(t *testing.T) {
	srv, store := newTestServer(t)
	url := srv.URL + "/events/" + testEvent + "/reserve"
	initial := store.eventsMap[testEvent]

	for want := 1; want <= initial; want++ {
		status, id, err := reserve(srv.Client(), url)
		if err != nil {
			t.Fatalf("request %d: %v", want, err)
		}
		if status != http.StatusCreated || id != want {
			t.Fatalf("request %d: got status %d id %d, want %d id %d",
				want, status, id, http.StatusCreated, want)
		}
	}

	status, _, err := reserve(srv.Client(), url)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusConflict {
		t.Errorf("after selling out: got status %d, want %d", status, http.StatusConflict)
	}
}

func TestReserveUnknownEvent(t *testing.T) {
	srv, _ := newTestServer(t)

	status, _, err := reserve(srv.Client(), srv.URL+"/events/nope/reserve")
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusNotFound {
		t.Errorf("got status %d, want %d", status, http.StatusNotFound)
	}
}

// TestConcurrentReservers fires many requests at once and checks invariants
// that must hold no matter how requests interleave. On the naive M1 store it
// is EXPECTED TO FAIL; that failure is the point of M1.
func TestConcurrentReservers(t *testing.T) {
	const requests = 1000

	srv, store := newTestServer(t)
	url := srv.URL + "/events/" + testEvent + "/reserve"
	initial := store.eventsMap[testEvent]

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
		sold     int        // even though the server under test isn't
		rejected int
		failed   int
		ids      = map[int]int{} // reservation id -> times it was issued
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
	remaining := store.eventsMap[testEvent]

	t.Logf("sold=%d rejected=%d failed=%d remaining=%d", sold, rejected, failed, remaining)

	if failed > 0 {
		t.Errorf("%d requests failed (transport error or unexpected status); counts are unreliable", failed)
	}

	// Invariant 1: never sell more tickets than exist.
	if sold > initial {
		t.Errorf("oversold: sold %d tickets, only %d exist", sold, initial)
	}

	// Invariant 2: every ticket is accounted for. If two requests read the
	// same count and both write count-1, both sell but one decrement is lost.
	if sold+remaining != initial {
		t.Errorf("lost updates: sold %d + remaining %d = %d, want %d",
			sold, remaining, sold+remaining, initial)
	}

	// Invariant 3: reservation ids are unique.
	for id, n := range ids {
		if n > 1 {
			t.Errorf("reservation id %d was issued %d times", id, n)
		}
	}
}
