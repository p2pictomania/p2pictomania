package web

import (
	"log"

	"github.com/gorilla/websocket"
)

// connection is an middleman between the websocket connection and the hub.
type connection struct {
	// The websocket connection.
	ws *websocket.Conn
	// Buffered channel of outbound messages.
	send chan []byte
}

// upgrader is used to upgrade connections to a websocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// write writes a message with the given message type and payload.
func (c *connection) write(messageType int, payload []byte) error {
	return c.ws.WriteMessage(messageType, payload)
}

// WriteMessagesToSocket writes the messages from the hub to the
// websocket connection.
func (c *connection) WriteMessagesToSocket() {
	// close socket if any errors occur in method
	defer func() {
		c.ws.Close()
	}()
	for {
		select {
		// if message was sent to the hub, get it and send it to the connection
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

// ReadMessagesFromSocket writes the messages from the
// websocket connection to the hub.
func (c *connection) ReadMessagesFromSocket() {
	// unregister the connection from the hub if there were errors in
	// reading from it and close the websocket connection
	defer func() {
		Hub.unregister <- c
		c.ws.Close()
	}()
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
		log.Printf("Got Message from client of length %d", len(message))
		// send recieved message to be broadcasted to all other
		// connections in the hub
		Hub.broadcast <- message
	}
}
