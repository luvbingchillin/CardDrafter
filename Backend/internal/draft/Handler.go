package draft

import (
	"backend/internal/middleware"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // Allows frontend dev server
    },
}

type DraftHandler struct {
	hub *Hub
}

func NewDraftHandler(hub *Hub) *DraftHandler {
	return &DraftHandler{hub: hub}
}

func (d *DraftHandler) ServeWs(w http.ResponseWriter, r *http.Request){
	roomId := r.PathValue("roomId")
	if roomId == ""{
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}
	//Read from cookie
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		// Fallback for development/testing if token cookie isn't set:
		userID = r.URL.Query().Get("userId")
		if userID == "" {
			http.Error(w, "Unauthorized: missing user identity", http.StatusUnauthorized)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err!=nil{
		log.Printf("Failed to upgrade websocket: %s", err)
		return
	}
	room := d.hub.GetOrCreateRoom(roomId)
	sendCh := make(chan []byte, 256)
	client := &Client{
		room:room,conn:conn, sendCh:sendCh,userID: userID,
	}
	room.register <- client
	go client.writePump()
	go client.readPump()

}