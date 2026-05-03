package ws_serv

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type NotifClient struct {
	UserID uint
	Conn   *websocket.Conn
	mu     sync.Mutex
}

func (c *NotifClient) SendJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteJSON(v)
}

// NotificationHub manages lightweight notification WebSocket connections.
type NotificationHub struct {
	clients map[uint]map[*NotifClient]bool
	mu      sync.RWMutex
}

func NewNotificationHub() *NotificationHub {
	return &NotificationHub{
		clients: make(map[uint]map[*NotifClient]bool),
	}
}

func (h *NotificationHub) Register(client *NotifClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*NotifClient]bool)
	}
	h.clients[client.UserID][client] = true
	zap.S().Debugf("notif client registered: userID=%d", client.UserID)
}

func (h *NotificationHub) Unregister(client *NotifClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[client.UserID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.clients, client.UserID)
		}
	}
	client.Conn.Close()
	zap.S().Debugf("notif client unregistered: userID=%d", client.UserID)
}

// Push sends a JSON message to all connected clients for the given user.
func (h *NotificationHub) Push(userID uint, v any) {
	h.mu.RLock()
	clients := h.clients[userID]
	h.mu.RUnlock()

	if len(clients) == 0 {
		return
	}

	data, err := json.Marshal(v)
	if err != nil {
		zap.S().Warnf("notif push marshal error: %s", err)
		return
	}

	for client := range clients {
		func(c *NotifClient) {
			c.mu.Lock()
			defer c.mu.Unlock()
			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				zap.S().Warnf("notif push write error userID=%d: %s", userID, err)
				c.Conn.Close()
				h.Unregister(c)
			}
		}(client)
	}
}

// BroadcastUnreadCount sends the unread count to all connected clients for the given user.
func (h *NotificationHub) BroadcastUnreadCount(userID uint, count int64) {
	h.Push(userID, map[string]any{
		"type":  "unread-count",
		"count": count,
	})
}
