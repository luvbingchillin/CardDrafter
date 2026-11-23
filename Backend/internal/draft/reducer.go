package draft

import (
	"backend/internal/models"
	"encoding/json"
	"log"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AggregateRoomState represents the pure in-memory state of a draft table,
// completely reconstructed by reducing domain events.
type AggregateRoomState struct {
	ID           string
	Version      int
	State        DraftState
	HostID       string
	PackSets     []string
	CurrentRound int
	Seats        []*Seat
}

// NewAggregateRoomState constructs an empty initial state before any events are applied.
func NewAggregateRoomState(roomID string) *AggregateRoomState {
	return &AggregateRoomState{
		ID:           roomID,
		Version:      0,
		State:        StateLobby,
		PackSets:     make([]string, 0),
		CurrentRound: 0,
		Seats:        make([]*Seat, 0),
	}
}

// Apply reduces a single immutable DraftEvent onto the AggregateRoomState.
// Pure function: no I/O, no network calls, completely deterministic.
func (s *AggregateRoomState) Apply(event DraftEvent) {
	s.Version = event.Version

	switch event.Type {
	case EventRoomCreated:
		var p RoomCreatedPayload
		if err := json.Unmarshal(event.Payload, &p); err == nil {
			s.HostID = p.HostID
			s.State = StateLobby
		}

	case EventPlayerJoined:
		var p PlayerJoinedPayload
		if err := json.Unmarshal(event.Payload, &p); err == nil {
			existingSeat := s.FindSeat(p.UserID)
			if existingSeat == nil {
				s.Seats = append(s.Seats, &Seat{
					Index:       p.SeatIndex,
					UserID:      p.UserID,
					IsReady:     false,
					CurrentPack: make([]*models.Card, 0),
					PackQueue:   make([][]*models.Card, 0),
					DraftedPool: make([]*models.Card, 0),
				})
			}
		}

	case EventPlayerLeft:
		var p PlayerLeftPayload
		if err := json.Unmarshal(event.Payload, &p); err == nil {
			if s.State == StateLobby {
				for i, seat := range s.Seats {
					if seat.UserID == p.UserID {
						s.Seats = append(s.Seats[:i], s.Seats[i+1:]...)
						for idx, remaining := range s.Seats {
							remaining.Index = idx
						}
						break
					}
				}
			}
		}

	case EventReadyToggled:
		var p ReadyToggledPayload
		if err := json.Unmarshal(event.Payload, &p); err == nil {
			seat := s.FindSeat(p.UserID)
			if seat != nil {
				seat.IsReady = p.IsReady
			}
		}

	case EventPacksSet:
		var p PacksSetPayload
		if err := json.Unmarshal(event.Payload, &p); err == nil {
			s.PackSets = p.PackSets
		}

	case EventRoundStarted:
		var p RoundStartedPayload
		if err := json.Unmarshal(event.Payload, &p); err == nil {
			s.State = StateDrafting
			s.CurrentRound = p.Round
			for seatIdx, pack := range p.SeatPacks {
				if seatIdx >= 0 && seatIdx < len(s.Seats) {
					seat := s.Seats[seatIdx]
					seat.CurrentPack = pack
					seat.TimeRemaining = 15 + len(pack)*2
					seat.PackQueue = make([][]*models.Card, 0)
				}
			}
		}

	case EventCardPicked:
		var p CardPickedPayload
		if err := json.Unmarshal(event.Payload, &p); err == nil {
			seat := s.FindSeat(p.UserID)
			if seat != nil && len(seat.CurrentPack) > 0 {
				cardID, err := bson.ObjectIDFromHex(p.CardID)
				if err == nil {
					for _, c := range seat.CurrentPack {
						if c.ID == cardID {
							seat.DraftedPool = append(seat.DraftedPool, c)
							// Advance the seat to its next queued pack (or nil if waiting on neighbor)
							if len(seat.PackQueue) > 0 {
								seat.CurrentPack = seat.PackQueue[0]
								seat.TimeRemaining = 15 + len(seat.CurrentPack)*2
								seat.PackQueue = seat.PackQueue[1:]
							} else {
								seat.CurrentPack = nil
								seat.TimeRemaining = 0
							}
							break
						}
					}
				}
			}
		}

	case EventPackPassed:
		var p PackPassedPayload
		if err := json.Unmarshal(event.Payload, &p); err == nil {
			if p.ToSeatIndex >= 0 && p.ToSeatIndex < len(s.Seats) {
				neighbor := s.Seats[p.ToSeatIndex]
				if len(neighbor.CurrentPack) == 0 {
					neighbor.CurrentPack = p.RemainingCards
					neighbor.TimeRemaining = 15 + len(p.RemainingCards)*2
				} else {
					neighbor.PackQueue = append(neighbor.PackQueue, p.RemainingCards)
				}
			}
		}

	case EventDraftCompleted:
		s.State = StateFinished

	default:
		log.Printf("Warning: unhandled event type in reducer: %s", event.Type)
	}
}

// FindSeat retrieves a seat pointer by UserID.
func (s *AggregateRoomState) FindSeat(userID string) *Seat {
	for _, seat := range s.Seats {
		if seat.UserID == userID {
			return seat
		}
	}
	return nil
}
