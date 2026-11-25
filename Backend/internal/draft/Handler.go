package draft

import (
	"backend/internal/middleware"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

func isOriginAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	// Allow same host or local development origins
	host := parsed.Hostname()
	if host == "localhost" || host == "127.0.0.1" || strings.EqualFold(parsed.Host, r.Host) {
		return true
	}
	if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
		if strings.EqualFold( strings.TrimRight(origin, "/"), strings.TrimRight(frontendURL, "/") ) {
			return true
		}
	}
	return false
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     isOriginAllowed,
}

type DraftHandler struct {
	hub *Hub
}

func NewDraftHandler(hub *Hub) *DraftHandler {
	return &DraftHandler{hub: hub}
}

func (d *DraftHandler) ServeWs(w http.ResponseWriter, r *http.Request) {
	roomId := r.PathValue("roomId")
	if roomId == "" {
		http.Error(w, `{"error":"room ID is required"}`, http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		http.Error(w, `{"error":"unauthorized: missing user identity"}`, http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("WebSocket upgrade failed", "room_id", roomId, "err", err)
		return
	}

	room := d.hub.GetOrCreateRoom(roomId)
	client := &Client{
		room:   room,
		conn:   conn,
		sendCh: make(chan []byte, 256),
		userID: userID,
	}

	// Safely register client; if room passivated concurrently, acquire a fresh room actor.
	select {
	case room.register <- client:
	case <-room.Done():
		room = d.hub.GetOrCreateRoom(roomId)
		client.room = room
		room.register <- client
	}

	go client.writePump()
	go client.readPump()
}
