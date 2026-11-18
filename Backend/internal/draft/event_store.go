package draft

import (
	"context"
)

// EventStore defines the persistence and retrieval contract for the draft event ledger.
// Adheres to Consumer-Driven Interface Segregation: the draft aggregate declares its storage contract.
type EventStore interface {
	// Append commits a new domain event to the room's stream.
	Append(ctx context.Context, roomID string, event DraftEvent) error

	// LoadEvents retrieves all domain events for a given room in chronological order.
	LoadEvents(ctx context.Context, roomID string) ([]DraftEvent, error)
}
