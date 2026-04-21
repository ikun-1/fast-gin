package signal_serv

import (
	"sync"
)

type Client struct {
	Conn   *Connection
	RoomNo string
	UserID uint
	Send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[uint]*Client
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[uint]*Client),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client.RoomNo]; !ok {
		h.clients[client.RoomNo] = make(map[uint]*Client)
	}
	h.clients[client.RoomNo][client.UserID] = client
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.clients[client.RoomNo]; ok {
		delete(room, client.UserID)
		if len(room) == 0 {
			delete(h.clients, client.RoomNo)
		}
	}
}

func (h *Hub) Broadcast(roomNo string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, ok := h.clients[roomNo]
	if !ok {
		return
	}
	for _, client := range room {
		select {
		case client.Send <- message:
		default:
		}
	}
}

func (h *Hub) ClientCount(roomNo string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[roomNo])
}
