package draft

import (
	"backend/internal/models"
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// inMemoryEventStore is a deterministic test double implementing EventStore.
type inMemoryEventStore struct {
	events  map[string][]DraftEvent
	failErr error
}

func newInMemoryEventStore() *inMemoryEventStore {
	return &inMemoryEventStore{
		events: make(map[string][]DraftEvent),
	}
}

func (m *inMemoryEventStore) Append(_ context.Context, roomID string, event DraftEvent) error {
	if m.failErr != nil {
		return m.failErr
	}
	m.events[roomID] = append(m.events[roomID], event)
	return nil
}

func (m *inMemoryEventStore) LoadEvents(_ context.Context, roomID string) ([]DraftEvent, error) {
	if m.failErr != nil {
		return nil, m.failErr
	}
	return m.events[roomID], nil
}

type mockPackOpener struct {
	packs map[string][]*models.Card
}

func (m *mockPackOpener) OpenPack(_ context.Context, setCode string, _ string) ([]*models.Card, error) {
	src := m.packs[setCode]
	out := make([]*models.Card, len(src))
	copy(out, src)
	return out, nil
}

func TestSingleSourceOfTruthAndRehydrationParity(t *testing.T) {
	store := newInMemoryEventStore()
	card1 := &models.Card{ID: bson.NewObjectID(), Name: "Lightning Bolt", Rarity: "common"}
	card2 := &models.Card{ID: bson.NewObjectID(), Name: "Counterspell", Rarity: "uncommon"}

	opener := &mockPackOpener{
		packs: map[string][]*models.Card{
			"FDN": {card1, card2},
		},
	}

	room := NewRoom("ROOM-ALPHA", opener, store, nil)

	// 1. Room creation & two players joining
	if err := room.emit(EventRoomCreated, RoomCreatedPayload{HostID: "player1"}); err != nil {
		t.Fatalf("emit EventRoomCreated: %v", err)
	}
	if err := room.emit(EventPlayerJoined, PlayerJoinedPayload{SeatIndex: 0, UserID: "player1"}); err != nil {
		t.Fatalf("emit EventPlayerJoined p1: %v", err)
	}
	if err := room.emit(EventPlayerJoined, PlayerJoinedPayload{SeatIndex: 1, UserID: "player2"}); err != nil {
		t.Fatalf("emit EventPlayerJoined p2: %v", err)
	}

	// 2. Ready toggles & pack configuration
	if err := room.emit(EventReadyToggled, ReadyToggledPayload{UserID: "player1", IsReady: true}); err != nil {
		t.Fatalf("emit EventReadyToggled p1: %v", err)
	}
	if err := room.emit(EventReadyToggled, ReadyToggledPayload{UserID: "player2", IsReady: true}); err != nil {
		t.Fatalf("emit EventReadyToggled p2: %v", err)
	}
	if err := room.emit(EventPacksSet, PacksSetPayload{PackSets: []string{"FDN"}}); err != nil {
		t.Fatalf("emit EventPacksSet: %v", err)
	}

	// 3. Round 0 starts with 2-card packs per seat
	p1Pack, _ := opener.OpenPack(context.Background(), "FDN", "player1")
	p2Pack, _ := opener.OpenPack(context.Background(), "FDN", "player2")
	if err := room.emit(EventRoundStarted, RoundStartedPayload{
		Round:   0,
		SetCode: "FDN",
		SeatPacks: map[int][]*models.Card{
			0: p1Pack,
			1: p2Pack,
		},
	}); err != nil {
		t.Fatalf("emit EventRoundStarted: %v", err)
	}

	// 4. Player 1 picks card1 -> remaining [card2] is passed to Player 2 (who still holds their pack, so it enters PackQueue)
	seat0 := room.agg.FindSeat("player1")
	room.executePick(seat0, 0, false)

	seat1 := room.agg.FindSeat("player2")
	if len(seat0.DraftedPool) != 1 || seat0.DraftedPool[0].ID != card1.ID {
		t.Fatalf("expected seat0 to have drafted card1")
	}
	if seat0.CurrentPack != nil {
		t.Fatalf("expected seat0 CurrentPack to be nil while waiting on neighbor, got %d cards", len(seat0.CurrentPack))
	}
	if len(seat1.PackQueue) != 1 || len(seat1.PackQueue[0]) != 1 {
		t.Fatalf("expected seat1 PackQueue to hold 1 passed pack with 1 card")
	}

	// 5. Simulate server crash & rehydrate a brand-new Room from the EventStore
	recoveredRoom := NewRoom("ROOM-ALPHA", opener, store, nil)
	recoveredRoom.recoverState()

	if recoveredRoom.agg.Version != room.agg.Version {
		t.Fatalf("version mismatch: live=%d recovered=%d", room.agg.Version, recoveredRoom.agg.Version)
	}
	if recoveredRoom.agg.State != StateDrafting {
		t.Fatalf("expected recovered state DRAFTING, got %s", recoveredRoom.agg.State)
	}
	recSeat0 := recoveredRoom.agg.FindSeat("player1")
	recSeat1 := recoveredRoom.agg.FindSeat("player2")
	if len(recSeat0.DraftedPool) != 1 || recSeat0.DraftedPool[0].ID != card1.ID {
		t.Fatalf("recovered seat0 drafted pool mismatch")
	}
	if recSeat0.CurrentPack != nil {
		t.Fatalf("recovered seat0 CurrentPack should be nil")
	}
	if len(recSeat1.PackQueue) != 1 || recSeat1.PackQueue[0][0].ID != card2.ID {
		t.Fatalf("recovered seat1 PackQueue mismatch")
	}
}

func TestStrictLedgerConsistencyAbortsOnAppendFailure(t *testing.T) {
	store := newInMemoryEventStore()
	room := NewRoom("ROOM-FAIL", nil, store, nil)

	// Force Redis Append failure
	store.failErr = errors.New("redis connection refused")

	err := room.emit(EventRoomCreated, RoomCreatedPayload{HostID: "host1"})
	if err == nil {
		t.Fatalf("expected emit to fail when EventStore.Append fails")
	}
	if room.agg.Version != 0 {
		t.Fatalf("expected version to remain 0 on failed Append, got %d", room.agg.Version)
	}
	if room.agg.HostID != "" {
		t.Fatalf("expected HostID to remain empty on failed Append, got %s", room.agg.HostID)
	}
}

func TestCompleteDraftTransition(t *testing.T) {
	store := newInMemoryEventStore()
	room := NewRoom("ROOM-DONE", nil, store, nil)

	_ = room.emit(EventRoomCreated, RoomCreatedPayload{HostID: "p1"})
	_ = room.emit(EventDraftCompleted, DraftCompletedPayload{CompletedAt: time.Now().UTC()})

	if room.agg.State != StateFinished {
		t.Fatalf("expected StateFinished, got %s", room.agg.State)
	}
}
