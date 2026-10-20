package draft

import (
	"encoding/json"
	"log"
)


type DraftState string

type ActionEnvelope struct {
    Type string `json:"type"`
}


type SetPackSetsMessage struct {
    Payload struct {
        PackSets []string `json:"packSets"`
    } `json:"payload"`
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
	Client *Client
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
}

func NewRoom(id string) *Room {
    return &Room{
        id:         id,
        state:      StateLobby,
        packSets:   []string{},
        seats:      make([]*Seat, 0),
        clients:    make(map[*Client]bool),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        actionChan: make(chan PlayerAction, 64),
    }
}


func (r *Room) Run() {
	for {
		select {
		case client := <-r.register:
		r.clients[client] = true

		// 1. Check if player is returning to an existing seat
		seat := r.findSeat(client.userID)
		if seat != nil {
			seat.Client = client // Reconnected!
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
						r.broadcastLobbyUpdate()
					}
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

