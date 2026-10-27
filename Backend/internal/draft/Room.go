package draft

import (
	"backend/internal/models"
	"context"
	"encoding/json"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)


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

type Seat struct{
	Index int
	UserID string
	IsReady bool
	Client *Client `json:"-"`
	CurrentPack []*models.Card `json:"-"`
	PackQueue   [][]*models.Card `json:"-"`
	DraftedPool []*models.Card `json:"-"`
	TimeRemaining int
}

type Room struct {
	unregister chan *Client
	actionChan chan PlayerAction
	id         string
	register   chan *Client
	clients    map[*Client]bool
	state DraftState
	hostID string
	packSets []string
	seats []*Seat
	packService PackOpener
	currentRound int
}

func NewRoom(id string, packService PackOpener) *Room {
    return &Room{
        id:         id,
        state:      StateLobby,
        packSets:   []string{},
        seats:      make([]*Seat, 0),
        clients:    make(map[*Client]bool),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        actionChan: make(chan PlayerAction, 64),
		packService: packService,
		currentRound: 0,
    }
}



func (r *Room) Run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if r.state == StateDrafting {
				r.handleTick()
			}

		case client := <-r.register:
		r.clients[client] = true

		// 1. Check if player is returning to an existing seat
		seat := r.findSeat(client.userID)
		if seat != nil {
			seat.Client = client // Reconnected!
			if r.state == StateDrafting{
				r.sendDraftUpdate(seat)
			}
		} else if r.state == StateLobby {
			// 2. New player joining the lobby
			if r.hostID == "" {
				r.hostID = client.userID
			}
			newSeat := &Seat{
				Index:   len(r.seats),
				UserID:  client.userID,
				Client:  client,
				IsReady: false,
			}
			r.seats = append(r.seats, newSeat)
		}

		// 3. Broadcast the updated lobby state to everyone!
		r.broadcastLobbyUpdate()

		case client := <-r.unregister:
			if _, ok := r.clients[client]; ok {
				delete(r.clients, client)
				close(client.sendCh)

				// Unlink the client from their seat, but keep the seat intact!
				seat := r.findSeat(client.userID)
				if seat != nil {
					seat.Client = nil
				}

				// Broadcast the update so other players see who went offline
				r.broadcastLobbyUpdate()
			}

		case action := <-r.actionChan:
			log.Printf("Draft room %s received action from player:%s:%s", r.id, action.UserID, string(action.Payload))
			envelope := ActionEnvelope{Type:""}
			err:=json.Unmarshal(action.Payload, &envelope)
			if err!=nil {
				log.Printf("Error occured")
				continue
			} else if r.state == StateLobby {
				switch envelope.Type {
				case "TOGGLE_READY":
					seat := r.findSeat(action.UserID)
					if seat != nil{
						seat.IsReady = !seat.IsReady
						r.broadcastLobbyUpdate()
					}
				case "SET_PACK_SETS":
					var setStore SetPackSetsMessage
					if action.UserID == r.hostID{
						err:=json.Unmarshal(action.Payload, &setStore)
						if err!=nil || len(setStore.Payload.PackSets)==0{
							log.Printf("Error occured")
							break
						}
						r.packSets = setStore.Payload.PackSets
						r.broadcastLobbyUpdate()
					}
				case "START_DRAFT":
					if action.UserID != r.hostID{
						log.Print("Not host")
						break
					}
					if r.state != StateLobby{
						log.Print("Invalid lobby state")
						break
					}
					if len(r.packSets) == 0 { log.Print("No packs configured"); break }
					valid:=true
					for _,seat:=range r.seats{
						if seat.Client!=nil{
							if !seat.IsReady{
								log.Print("User not ready")
								valid = false
								break
							}
						}
					}
					if valid && len(r.seats)>0{
						r.state = StateDrafting
						r.currentRound = 0
						currentSet := r.packSets[r.currentRound]
						for _,seat:=range r.seats{
							pack, err:= r.packService.OpenPack(context.Background(), currentSet, seat.UserID)
							if err != nil{
								log.Print("Invalid pack")
							}
							seat.CurrentPack = pack
							seat.TimeRemaining = 15 + len(pack)*2
							seat.PackQueue = make([][]*models.Card, 0)
							seat.DraftedPool = make([]*models.Card, 0)
							r.sendDraftUpdate(seat)
						}
						r.broadcastLobbyUpdate()
					}
				default:
					log.Printf("unknown input %s", envelope.Type)
				}
			} else if r.state == StateDrafting {
				seat := r.findSeat(action.UserID)
				if seat == nil || len(seat.CurrentPack) == 0 {
					log.Print("Invalid seat or empty pack")
					continue
				}
				switch envelope.Type {
				case "PICK_CARD":
					r.handlePickCard(seat, action.Payload)
				default:
					log.Printf("unknown input %s", envelope.Type)
				}
			}
		}
	}
}

func (r *Room) broadcast(msg WSMessage){
	message,err := json.Marshal(msg)
	if err!=nil{
		log.Printf("Invalid message %s", err)
		return 
	}
	for client := range r.clients{
		select{
		case client.sendCh<-message:
		default:
			log.Printf("Warning: Client %s buffer full; dropped message", client.userID)
		}
		}
	}

func (r *Room) findSeat(userID string) *Seat {
	for _, seat := range r.seats {
		if seat.UserID == userID {
			return seat
		}
	}
	return nil // Not found
}


func (r *Room) broadcastLobbyUpdate(){
	r.broadcast(WSMessage{
		Type: "LOBBY_UPDATE",
		Payload: map[string]any{
			"hostId":   r.hostID,
			"seats":    r.seats,
			"packSets": r.packSets,
			"state":    r.state,
		},
	})
}

func (r *Room) sendToClient(client *Client, msg WSMessage){
	if client == nil{
		return
	}
	message, err := json.Marshal(msg)
	if err!=nil{
		log.Printf("Invalid message %s", err)
		return 
	}
	select {
	case client.sendCh <- message:
	default:
		log.Printf("Warning: Client %s buffer full; dropped message", client.userID)
	}
}


func (r *Room) sendDraftUpdate(seat *Seat){
	if seat.Client == nil{
		return
	}
	r.sendToClient(seat.Client, WSMessage{
		Type: "DRAFT_UPDATE",
		Payload: map[string]any{
            "round":       r.currentRound + 1,
            "totalRounds": len(r.packSets),
            "currentPack": seat.CurrentPack,
            "draftedPool": seat.DraftedPool,
            "queueCount":  len(seat.PackQueue),
			"timeRemaining": seat.TimeRemaining,
		},
		
	})
}

func (r *Room) getNeighborSeat(currentIndex int) *Seat {
	total := len(r.seats)
	if total == 0 {
		return nil
	}
	var neighborIndex int
	if r.currentRound%2 == 0 {
		// Round 0 (Pack 1), Round 2 (Pack 3): Pass Left (Clockwise)
		neighborIndex = (currentIndex + 1) % total
	} else {
		// Round 1 (Pack 2): Pass Right (Counter-Clockwise)
		neighborIndex = (currentIndex - 1 + total) % total
	}
	return r.seats[neighborIndex]
}

func (r *Room) passPack(fromSeat *Seat, pack []*models.Card) {
	if len(pack) == 0 {
		return // Pack is exhausted; last card was drafted
	}
	neighbor := r.getNeighborSeat(fromSeat.Index)
	if neighbor == nil {
		return
	}
	if len(neighbor.CurrentPack) == 0 {
		neighbor.CurrentPack = pack
	} else {
		neighbor.PackQueue = append(neighbor.PackQueue, pack)
	}
	r.sendDraftUpdate(neighbor)
}

func (r *Room) advanceSeatPack(seat *Seat) {
	if len(seat.PackQueue) > 0 {
		seat.CurrentPack = seat.PackQueue[0]
		seat.TimeRemaining = 15 + len(seat.CurrentPack)*2
		seat.PackQueue = seat.PackQueue[1:]
	} else {
		seat.CurrentPack = nil
	}
	r.sendDraftUpdate(seat)
}

func (r *Room) checkRoundCompletion() {
	for _, s := range r.seats {
		if len(s.CurrentPack) > 0 || len(s.PackQueue) > 0 {
			return // Still drafting this round
		}
	}

	// All players finished drafting their packs for this round!
	r.currentRound++
	if r.currentRound < len(r.packSets) {
		currentSet := r.packSets[r.currentRound]
		for _, s := range r.seats {
			pack, err := r.packService.OpenPack(context.Background(), currentSet, s.UserID)
			if err != nil {
				log.Printf("Error opening pack for seat %d: %v", s.Index, err)
			}
			s.CurrentPack = pack
			s.TimeRemaining = 15 + len(s.CurrentPack)*2
			s.PackQueue = make([][]*models.Card, 0)
			r.sendDraftUpdate(s)
		}
		r.broadcastLobbyUpdate()
	} else {
		r.state = StateFinished
		r.broadcastLobbyUpdate()
	}
}

func (r *Room) handlePickCard(seat *Seat, payload []byte) {
	var msg PickCardMessage
	if err := json.Unmarshal(payload, &msg); err != nil || msg.Payload.CardID == "" {
		log.Print("Invalid card payload")
		return
	}

	cardID, err := bson.ObjectIDFromHex(msg.Payload.CardID)
	if err != nil {
		log.Print("Unable to convert object from hex")
		return
	}

	cardFound := false
	for i, card := range seat.CurrentPack {
		if card.ID == cardID {
			r.executePick(seat, i)
			cardFound = true
			break
		}
	}

	if !cardFound {
		log.Printf("Card %s not found in current pack for seat %d", msg.Payload.CardID, seat.Index)
		return
	}
}


func (r *Room) handleTick(){
	for _,seat:=range r.seats{
		if len(seat.CurrentPack)>0{
			seat.TimeRemaining-=1
			if seat.TimeRemaining<=0{
				r.executePick(seat, 0)

			} else {
				r.sendToClient(seat.Client, WSMessage{Type:"TIMER_TICK", Payload: map[string]any{"timeRemaining":seat.TimeRemaining}})
			}
		}
	}
}

func (r *Room) executePick(seat *Seat, cardIndex int){
	card := seat.CurrentPack[cardIndex]
	seat.DraftedPool = append(seat.DraftedPool, card)
	seat.CurrentPack = append(seat.CurrentPack[:cardIndex], seat.CurrentPack[cardIndex+1:]...)
	r.passPack(seat, seat.CurrentPack)
	r.advanceSeatPack(seat)
	r.checkRoundCompletion()
}