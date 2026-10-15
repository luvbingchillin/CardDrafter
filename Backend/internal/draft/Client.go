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
	conn   *websocket.Conn
	sendCh chan []byte
	userID string
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
