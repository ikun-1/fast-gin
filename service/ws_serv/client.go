package ws_serv

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

type Client struct {
	Hub         *Hub
	Conn        *websocket.Conn
	Send        chan []byte
	UserID      uint
	ClientID    string
	RoomNo      uint
	MeetingID   uint
	DisplayName string
	Avatar      string
	IsHost      bool
	mu          sync.Mutex

	// Pion WebRTC PeerConnection (SFU-side)
	PC      *webrtc.PeerConnection
	pcReady sync.Once
	pcMu    sync.Mutex

	// Tracks this client is sending to the server (remote tracks)
	// Stored in the Room's track registry, not here directly.
	// This field tracks the local tracks we created FOR others.
	LocalTracks []*webrtc.TrackLocalStaticRTP

	// Negotiation
	negotiateChan chan struct{}
}

var clientIDCounter uint64

func NewClient(hub *Hub, conn *websocket.Conn, userID uint, displayName, avatar string) *Client {
	return &Client{
		Hub:           hub,
		Conn:          conn,
		Send:          make(chan []byte, 256),
		UserID:        userID,
		ClientID:      fmt.Sprintf("user_%d_%d", userID, atomic.AddUint64(&clientIDCounter, 1)),
		DisplayName:   displayName,
		Avatar:        avatar,
		negotiateChan: make(chan struct{}, 1),
	}
}

func (c *Client) CreatePeerConnection(config webrtc.Configuration) (*webrtc.PeerConnection, error) {
	c.pcMu.Lock()
	defer c.pcMu.Unlock()

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("create peer connection: %w", err)
	}

	c.PC = pc

	// Handle ICE candidates
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return // All candidates gathered
		}
		init := candidate.ToJSON()
		c.SendJSON(WsServerMessage{
			Type: "ice-candidate",
			Data: init,
		})
	})

	// Handle connection state changes
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		zap.S().Infof("PC state changed client=%s state=%s", c.ClientID, state)
		if state == webrtc.PeerConnectionStateDisconnected ||
			state == webrtc.PeerConnectionStateFailed {
			c.Hub.Unregister <- c
		}
	})

	// When the server needs to renegotiate (e.g., adding forwarded tracks to this PC)
	pc.OnNegotiationNeeded(func() {
		select {
		case c.negotiateChan <- struct{}{}:
		default:
		}
	})

	return pc, nil
}

func removeDuplicateMsidLines(sdp string) string {
	lines := strings.Split(sdp, "\r\n")
	seen := make(map[string]bool)
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "a=msid:") {
			if seen[line] {
				continue
			}
			seen[line] = true
		}
		result = append(result, line)
	}
	return strings.Join(result, "\r\n")
}

func (c *Client) HandleNegotiation() {
	if c.PC == nil {
		return
	}

	offer, err := c.PC.CreateOffer(nil)
	if err != nil {
		zap.S().Errorf("CreateOffer for renegotiation failed client=%s: %s", c.ClientID, err)
		return
	}

	// Strip duplicate a=msid lines to avoid browser SDP parsing errors
	offer.SDP = removeDuplicateMsidLines(offer.SDP)

	if err := c.PC.SetLocalDescription(offer); err != nil {
		zap.S().Errorf("SetLocalDescription for renegotiation failed client=%s: %s", c.ClientID, err)
		return
	}

	c.SendJSON(WsServerMessage{
		Type: "offer",
		Data: RenegotiationOffer{SDP: offer.SDP},
	})
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				zap.S().Warnf("WebSocket read error client=%s: %s", c.ClientID, err)
			}
			break
		}

		var msg WsClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			zap.S().Warnf("Invalid message from client=%s: %s", c.ClientID, err)
			continue
		}

		c.Hub.HandleMessage(c, &msg)
	}
}

func (c *Client) WritePump() {
	defer c.Conn.Close()

	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			zap.S().Warnf("WebSocket write error client=%s: %s", c.ClientID, err)
			break
		}
	}
}

func (c *Client) SendJSON(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		zap.S().Errorf("Marshal message error: %s", err)
		return
	}
	select {
	case c.Send <- data:
	default:
		zap.S().Warnf("Client send buffer full, dropping message client=%s", c.ClientID)
	}
}
