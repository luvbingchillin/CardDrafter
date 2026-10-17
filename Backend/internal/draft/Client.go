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
	// (9/10th of 60s = 54 seconds)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer (512 bytes is plenty for card pick IDs).
	maxMessageSize = 512
)


type Client struct {
	room *Room
	conn   *websocket.Conn
	sendCh chan []byte
	userID string
}

type PlayerAction struct {
	UserID  string
	Payload []byte // The raw JSON message from the client (e.g. {"type":"PICK_CARD","card_id":"..."})
}



// writePump pumps messages from the sendCh to the websocket connection.
// Exactly ONE writePump goroutine runs per active connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.sendCh:
			// 1. Set a write timeout: socket must accept write within 10 seconds!
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The channel was closed by the server (e.g. room shut down or kick)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 2. Write the message to the socket
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return // Network error! defer will clean up the connection
			}

		case <-ticker.C:
			// 3. 54 seconds elapsed with no action: send a Ping heartbeat
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return // Client didn't respond or socket broken! Kill goroutine
			}
		}
	}
}


// readPump pumps incoming messages from the websocket connection to the room's action channel.
// Exactly ONE readPump goroutine runs per active connection.
func (c *Client) readPump() {
	defer func() {
		// When the connection dies, notify the room to unregister this client!
		c.room.unregister <- c
		c.conn.Close()
	}()

	// 1. Defend against malicious clients sending giant payloads (max 512 bytes)
	c.conn.SetReadLimit(maxMessageSize)

	// 2. Set the initial read deadline to 60 seconds from now
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	// 3. Every time the browser replies with a Pong, push the deadline back another 60s!
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// 4. Infinite read loop
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			// Normal disconnects (e.g. closing the browser tab) trigger an error here
			break // Exits loop, triggering the defer cleanup above
		}

		// Forward the player's action into the room's central channel!
		c.room.actionChan <- PlayerAction{
			UserID:  c.userID,
			Payload: message,
		}
	}
}
