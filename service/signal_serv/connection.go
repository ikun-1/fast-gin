package signal_serv

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

type Connection struct {
	WS     *websocket.Conn
	Send   chan []byte
	Hub    *Hub
	RoomNo string
	UserID uint
}

func NewConnection(ws *websocket.Conn, hub *Hub, roomNo string, userID uint) *Connection {
	return &Connection{
		WS:     ws,
		Send:   make(chan []byte, 32),
		Hub:    hub,
		RoomNo: roomNo,
		UserID: userID,
	}
}

func (c *Connection) Start() {
	go c.writePump()
	c.readPump()
}

func (c *Connection) readPump() {
	defer func() {
		c.Hub.Unregister(&Client{Conn: c, RoomNo: c.RoomNo, UserID: c.UserID, Send: c.Send})
		_ = c.WS.Close()
	}()

	for {
		_, message, err := c.WS.ReadMessage()
		if err != nil {
			return
		}
		c.Hub.Broadcast(c.RoomNo, message)
	}
}

func (c *Connection) writePump() {
	defer func() {
		_ = c.WS.Close()
	}()

	for msg := range c.Send {
		_ = c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.WS.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *Connection) SendJSON(v any) error {
	msg, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.Send <- msg
	return nil
}
