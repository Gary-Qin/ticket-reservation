// Package events handles routes & services for /events/
package events

type Reserver interface {
	Reserve(id string) error
}
