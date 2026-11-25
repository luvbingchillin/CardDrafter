package draft

import (
	"backend/internal/models"
	"context"
	"encoding/json"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const roomIdleTimeout = 5 * time.Minute

type DraftState string

type ActionEnvelope struct {
	Type string `json:"type"`
}

type PickCardMessage struct {
	Payload struct {
		CardID string `json:"cardId"`
	} `json:"payload"`
}

type SetPackSetsMessage struct {
	Payload struct {
		PackSets []string `json:"packSets"`
	} `json:"payload"`
}

type PackOpener interface {
	OpenPack(ctx context.Context, setCode string, userID string) ([]*models.Card, error)
}

const (
	StateLobby    DraftState = "LOBBY"
	StateDrafting DraftState = "DRAFTING"
	StateFinished DraftState = "FINISHED"
)

type WSMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type Seat struct {
	Index         int
	UserID        string
	IsReady       bool
	IsConnected   bool
	Client        *Client          `json:"-"`
	CurrentPack   []*models.Card   `json:"-"`
	PackQueue     [][]*models.Card `json:"-"`
	DraftedPool   []*models.Card   `json:"-"`
	TimeRemaining int
}

type Room struct {
	id          string
	agg         *AggregateRoomState
	hub         *Hub
	packService PackOpener
	eventStore  EventStore
	clients     map[*Client]bool
	register    chan *Client
	unregister  chan *Client
	actionChan  chan PlayerAction
	done        chan struct{}
}

func NewRoom(id string, packService PackOpener, eventStore EventStore, hub *Hub) *Room {
	return &Room{
		id:          id,
		agg:         NewAggregateRoomState(id),
		hub:         hub,
		packService: packService,
		eventStore:  eventStore,
		clients:     make(map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		actionChan:  make(chan PlayerAction, 64),
		done:        make(chan struct{}),
	}
}

// Done returns a channel that is closed when the Room actor loop terminates.
func (r *Room) Done() <-chan struct{} {
	return r.done
}

func (r *Room) Run() {
	defer close(r.done)
	r.recoverState()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	idleTimer := time.NewTimer(roomIdleTimeout)
	defer idleTimer.Stop()

	for {
		select {
		case <-ticker.C:
			if r.agg.State == StateDrafting {
				r.handleTick()
			}

		case <-idleTimer.C:
			if len(r.clients) == 0 {
				log.Printf("Room %s idle for %v with 0 connected clients; passivating actor", r.id, roomIdleTimeout)
				if r.hub != nil {
					r.hub.RemoveRoom(r.id)
				}
				return
			}

		case client := <-r.register:
			// Stop idle passivation timer since a client is now active
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}

			r.clients[client] = true

			// 1. Check if player is returning to an existing seat
			seat := r.agg.FindSeat(client.userID)
			if seat != nil {
				// Clean up any stale socket connection if user opened a duplicate tab
				if seat.Client != nil && seat.Client != client {
					if _, ok := r.clients[seat.Client]; ok {
						delete(r.clients, seat.Client)
						close(seat.Client.sendCh)
					}
				}
				seat.Client = client
				seat.IsConnected = true

				// Send private draft state in both DRAFTING and FINISHED states so deck pools persist across refresh
				if r.agg.State == StateDrafting || r.agg.State == StateFinished {
					r.sendDraftUpdate(seat)
				}
			} else if r.agg.State == StateLobby {
				// 2. New player joining the lobby
				if r.agg.HostID == "" {
					if err := r.emit(EventRoomCreated, RoomCreatedPayload{HostID: client.userID}); err != nil {
						continue
					}
				}
				seatIndex := len(r.agg.Seats)
				if err := r.emit(EventPlayerJoined, PlayerJoinedPayload{
					SeatIndex: seatIndex,
					UserID:    client.userID,
				}); err != nil {
					continue
				}
				if newSeat := r.agg.FindSeat(client.userID); newSeat != nil {
					newSeat.Client = client
					newSeat.IsConnected = true
				}
			}

			// 3. Broadcast the updated lobby state to everyone
			r.broadcastLobbyUpdate()

		case client := <-r.unregister:
			if _, ok := r.clients[client]; ok {
				delete(r.clients, client)
				close(client.sendCh)

				seat := r.agg.FindSeat(client.userID)
				if seat != nil && seat.Client == client {
					seat.Client = nil
					seat.IsConnected = false
				}

				r.broadcastLobbyUpdate()

				// Start 5-minute idle passivation timer when room becomes empty
				if len(r.clients) == 0 {
					idleTimer.Reset(roomIdleTimeout)
				}
			}

		case action := <-r.actionChan:
			log.Printf("Draft room %s received action from player %s: %s", r.id, action.UserID, string(action.Payload))
			var envelope ActionEnvelope
			if err := json.Unmarshal(action.Payload, &envelope); err != nil {
				log.Printf("Invalid action envelope in room %s: %v", r.id, err)
				continue
			}

			if r.agg.State == StateLobby {
				switch envelope.Type {
				case "TOGGLE_READY":
					seat := r.agg.FindSeat(action.UserID)
					if seat != nil {
						if err := r.emit(EventReadyToggled, ReadyToggledPayload{
							UserID:  seat.UserID,
							IsReady: !seat.IsReady,
						}); err == nil {
							r.broadcastLobbyUpdate()
						}
					}

				case "SET_PACK_SETS":
					if action.UserID == r.agg.HostID {
						var setStore SetPackSetsMessage
						if err := json.Unmarshal(action.Payload, &setStore); err != nil || len(setStore.Payload.PackSets) == 0 {
							log.Printf("Invalid SET_PACK_SETS payload in room %s", r.id)
							break
						}
						if err := r.emit(EventPacksSet, PacksSetPayload{
							PackSets: setStore.Payload.PackSets,
						}); err == nil {
							r.broadcastLobbyUpdate()
						}
					}

				case "START_DRAFT":
					if action.UserID != r.agg.HostID {
						log.Print("Unauthorized START_DRAFT: user is not host")
						break
					}
					if len(r.agg.PackSets) == 0 {
						log.Print("Cannot start draft: no packs configured")
						break
					}
					valid := true
					for _, seat := range r.agg.Seats {
						if seat.IsConnected && !seat.IsReady {
							log.Printf("Cannot start draft: user %s is not ready", seat.UserID)
							valid = false
							break
						}
					}
					if valid && len(r.agg.Seats) > 0 {
						currentSet := r.agg.PackSets[0]
						seatPacks := make(map[int][]*models.Card, len(r.agg.Seats))
						for _, seat := range r.agg.Seats {
							pack, err := r.packService.OpenPack(context.Background(), currentSet, seat.UserID)
							if err != nil {
								log.Printf("Error opening initial pack for seat %d: %v", seat.Index, err)
							}
							seatPacks[seat.Index] = pack
						}
						if err := r.emit(EventRoundStarted, RoundStartedPayload{
							Round:     0,
							SetCode:   currentSet,
							SeatPacks: seatPacks,
						}); err != nil {
							break
						}
						for _, seat := range r.agg.Seats {
							r.sendDraftUpdate(seat)
						}
						r.broadcastLobbyUpdate()
					}

				default:
					log.Printf("Unknown lobby action type: %s", envelope.Type)
				}
			} else if r.agg.State == StateDrafting {
				seat := r.agg.FindSeat(action.UserID)
				if seat == nil || len(seat.CurrentPack) == 0 {
					log.Print("Invalid seat or empty current pack")
					continue
				}
				switch envelope.Type {
				case "PICK_CARD":
					r.handlePickCard(seat, action.Payload)
				default:
					log.Printf("Unknown drafting action type: %s", envelope.Type)
				}
			}
		}
	}
}

func (r *Room) broadcast(msg WSMessage) {
	message, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Invalid broadcast message: %v", err)
		return
	}
	for client := range r.clients {
		select {
		case client.sendCh <- message:
		default:
			log.Printf("Warning: Client %s buffer full; dropped message", client.userID)
		}
	}
}

func (r *Room) broadcastLobbyUpdate() {
	r.broadcast(WSMessage{
		Type: "LOBBY_UPDATE",
		Payload: map[string]any{
			"hostId":   r.agg.HostID,
			"seats":    r.agg.Seats,
			"packSets": r.agg.PackSets,
			"state":    r.agg.State,
		},
	})
}

func (r *Room) sendToClient(client *Client, msg WSMessage) {
	if client == nil {
		return
	}
	message, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Invalid client message: %v", err)
		return
	}
	select {
	case client.sendCh <- message:
	default:
		log.Printf("Warning: Client %s buffer full; dropped message", client.userID)
	}
}

func (r *Room) sendDraftUpdate(seat *Seat) {
	if seat.Client == nil {
		return
	}
	r.sendToClient(seat.Client, WSMessage{
		Type: "DRAFT_UPDATE",
		Payload: map[string]any{
			"round":         r.agg.CurrentRound + 1,
			"totalRounds":   len(r.agg.PackSets),
			"currentPack":   seat.CurrentPack,
			"draftedPool":   seat.DraftedPool,
			"queueCount":    len(seat.PackQueue),
			"timeRemaining": seat.TimeRemaining,
		},
	})
}

func (r *Room) getNeighborSeat(currentIndex int) *Seat {
	total := len(r.agg.Seats)
	if total == 0 {
		return nil
	}
	var neighborIndex int
	if r.agg.CurrentRound%2 == 0 {
		// Round 0 (Pack 1), Round 2 (Pack 3): Pass Left (Clockwise)
		neighborIndex = (currentIndex + 1) % total
	} else {
		// Round 1 (Pack 2): Pass Right (Counter-Clockwise)
		neighborIndex = (currentIndex - 1 + total) % total
	}
	return r.agg.Seats[neighborIndex]
}

func (r *Room) checkRoundCompletion() {
	for _, s := range r.agg.Seats {
		if len(s.CurrentPack) > 0 || len(s.PackQueue) > 0 {
			return // Still drafting this round
		}
	}

	// All players finished drafting their packs for this round
	nextRound := r.agg.CurrentRound + 1
	if nextRound < len(r.agg.PackSets) {
		currentSet := r.agg.PackSets[nextRound]
		seatPacks := make(map[int][]*models.Card, len(r.agg.Seats))
		for _, s := range r.agg.Seats {
			pack, err := r.packService.OpenPack(context.Background(), currentSet, s.UserID)
			if err != nil {
				log.Printf("Error opening pack for seat %d: %v", s.Index, err)
			}
			seatPacks[s.Index] = pack
		}
		if err := r.emit(EventRoundStarted, RoundStartedPayload{
			Round:     nextRound,
			SetCode:   currentSet,
			SeatPacks: seatPacks,
		}); err != nil {
			return
		}
		for _, s := range r.agg.Seats {
			r.sendDraftUpdate(s)
		}
		r.broadcastLobbyUpdate()
	} else {
		if err := r.emit(EventDraftCompleted, DraftCompletedPayload{
			CompletedAt: time.Now().UTC(),
		}); err != nil {
			return
		}
		r.broadcastLobbyUpdate()
	}
}

func (r *Room) handlePickCard(seat *Seat, payload []byte) {
	var msg PickCardMessage
	if err := json.Unmarshal(payload, &msg); err != nil || msg.Payload.CardID == "" {
		log.Print("Invalid PICK_CARD payload")
		return
	}

	cardID, err := bson.ObjectIDFromHex(msg.Payload.CardID)
	if err != nil {
		log.Print("Unable to parse card ObjectID from hex")
		return
	}

	cardFound := false
	for i, card := range seat.CurrentPack {
		if card.ID == cardID {
			r.executePick(seat, i, false)
			cardFound = true
			break
		}
	}

	if !cardFound {
		log.Printf("Card %s not found in current pack for seat %d", msg.Payload.CardID, seat.Index)
	}
}

func (r *Room) handleTick() {
	for _, seat := range r.agg.Seats {
		if len(seat.CurrentPack) > 0 {
			seat.TimeRemaining--
			if seat.TimeRemaining <= 0 {
				r.executePick(seat, 0, true)
			} else {
				r.sendToClient(seat.Client, WSMessage{
					Type:    "TIMER_TICK",
					Payload: map[string]any{"timeRemaining": seat.TimeRemaining},
				})
			}
		}
	}
}

func (r *Room) executePick(seat *Seat, cardIndex int, isAutoPick bool) {
	card := seat.CurrentPack[cardIndex]

	// Copy remaining cards into a fresh slice before emitting EventCardPicked
	remaining := make([]*models.Card, 0, len(seat.CurrentPack)-1)
	for i, c := range seat.CurrentPack {
		if i != cardIndex {
			remaining = append(remaining, c)
		}
	}

	if err := r.emit(EventCardPicked, CardPickedPayload{
		SeatIndex:  seat.Index,
		UserID:     seat.UserID,
		CardID:     card.ID.Hex(),
		IsAutoPick: isAutoPick,
	}); err != nil {
		return
	}

	if len(remaining) > 0 {
		neighbor := r.getNeighborSeat(seat.Index)
		if neighbor != nil {
			if err := r.emit(EventPackPassed, PackPassedPayload{
				FromSeatIndex:  seat.Index,
				ToSeatIndex:    neighbor.Index,
				RemainingCards: remaining,
			}); err != nil {
				return
			}
			r.sendDraftUpdate(neighbor)
		}
	}

	r.sendDraftUpdate(seat)
	r.checkRoundCompletion()
}

// emit enforces strict event-ledger consistency:
// 1. Serializes the domain payload and constructs the next sequentially versioned DraftEvent.
// 2. Appends to Redis first; if Redis fails, aborts before mutating in-memory AggregateRoomState.
// 3. Applies the event onto r.agg as the single source of truth.
func (r *Room) emit(eventType EventType, payload any) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal event payload (%s): %v", eventType, err)
		return err
	}

	nextVersion := r.agg.Version + 1
	event := DraftEvent{
		RoomID:    r.id,
		Type:      eventType,
		Version:   nextVersion,
		Timestamp: time.Now().UTC(),
		Payload:   jsonBytes,
	}

	if r.eventStore != nil {
		if err := r.eventStore.Append(context.Background(), r.id, event); err != nil {
			log.Printf("Failed to append event %s (v%d) to Redis for room %s: %v", eventType, nextVersion, r.id, err)
			return err
		}
	}

	r.agg.Apply(event)
	return nil
}

func (r *Room) recoverState() {
	if r.eventStore == nil {
		return
	}
	events, err := r.eventStore.LoadEvents(context.Background(), r.id)
	if err != nil {
		log.Printf("Failed to load historical events for room %s: %v", r.id, err)
		return
	}
	if len(events) == 0 {
		return
	}

	agg := NewAggregateRoomState(r.id)
	for _, ev := range events {
		agg.Apply(ev)
	}
	r.agg = agg
	log.Printf("Room %s rehydrated to version %d (%d events replayed)", r.id, r.agg.Version, len(events))
}
