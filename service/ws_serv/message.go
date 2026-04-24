package ws_serv

import "encoding/json"

// ---------- Client -> Server ----------

type WsClientMessage struct {
	Type string `json:"type"`
	// join-room
	RoomNo   uint   `json:"roomNo,omitempty"`
	Password string `json:"password,omitempty"`
	// offer / answer
	SDP json.RawMessage `json:"sdp,omitempty"`
	// ice-candidate
	Candidate json.RawMessage `json:"candidate,omitempty"`
	// mute-toggle
	Muted bool   `json:"muted,omitempty"`
	Kind  string `json:"kind,omitempty"`
	// chat-message
	Text string `json:"text,omitempty"`
}

// ---------- Server -> Client ----------

type WsServerMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

type ParticipantInfo struct {
	ClientID    string `json:"clientId"`
	DisplayName string `json:"displayName"`
	IsHost      bool   `json:"isHost"`
	IsMuted     bool   `json:"isMuted"`
	IsCamOff    bool   `json:"isCamOff"`
}

type RoomJoinedData struct {
	RoomNo       uint              `json:"roomNo"`
	ClientID     string            `json:"clientId"`
	Participants []ParticipantInfo `json:"participants"`
}

type UserJoinedData struct {
	ClientID    string `json:"clientId"`
	DisplayName string `json:"displayName"`
	IsHost      bool   `json:"isHost"`
}

type UserLeftData struct {
	ClientID string `json:"clientId"`
}

type ForwardedSDP struct {
	FromClientID string          `json:"fromClientId"`
	SDP          json.RawMessage `json:"sdp"`
}

type ForwardedCandidate struct {
	FromClientID string          `json:"fromClientId"`
	Candidate    json.RawMessage `json:"candidate"`
}

type MuteToggleData struct {
	ClientID string `json:"clientId"`
	Muted    bool   `json:"muted"`
	Kind     string `json:"kind"`
}

type ScreenShareData struct {
	ClientID string `json:"clientId"`
}

type ChatMessageData struct {
	FromClientID string `json:"fromClientId"`
	DisplayName  string `json:"displayName"`
	Text         string `json:"text"`
}

// Server-initiated SDP offer for renegotiation (forwarded track from another client)
type RenegotiationOffer struct {
	SDP string `json:"sdp"`
}
