package draft

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait!
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer (512 bytes is plenty for card pick IDs).
	maxMessageSize = 512
)

type Client struct {
	room   *Room
	conn   *websocket.Conn
	sendCh chan []byte
	userID string
}

type PlayerAction struct {
	UserID  string
	Payload []byte
}

// writePump pumps messages from sendCh to the websocket connection.
// Exactly ONE writePump goroutine runs per active connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.sendCh:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps incoming messages from the websocket connection to the room's action channel.
// Exactly ONE readPump goroutine runs per active connection.
func (c *Client) readPump() {
	defer func() {
		// Guard against blocking if the room actor has already passivated
		select {
		case c.room.unregister <- c:
		case <-c.room.Done():
		}
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		select {
		case c.room.actionChan <- PlayerAction{
			UserID:  c.userID,
			Payload: message,
		}:
		case <-c.room.Done():
			return
		}
	}
}
