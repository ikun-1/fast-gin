package ws_serv

import (
	"encoding/json"
	"time"
)

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
	// recording-control
	Action string `json:"action,omitempty"`
	// quality-report
	Metrics json.RawMessage `json:"metrics,omitempty"`
}

// ---------- Server -> Client ----------

type WsServerMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

type ParticipantInfo struct {
	ClientID    string `json:"clientId"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar,omitempty"`
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
	Avatar      string `json:"avatar,omitempty"`
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

// Recording control data (server -> client)
type RecordingControlData struct {
	Action     string `json:"action"` // "started" or "stopped"
	StartedAt  string `json:"startedAt,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
}

// Quality metric snapshot from client (browser getStats)
type QualityMetricSnapshot struct {
	Label          string    `json:"label"`                    // "audio", "video", "connection"
	BytesSent      int64     `json:"bytesSent,omitempty"`
	BytesReceived  int64     `json:"bytesReceived,omitempty"`
	PacketsSent    int64     `json:"packetsSent,omitempty"`
	PacketsReceived int64    `json:"packetsReceived,omitempty"`
	PacketsLost    int64     `json:"packetsLost,omitempty"`
	JitterMs       float64   `json:"jitterMs,omitempty"`
	RoundTripMs    float64   `json:"roundTripMs,omitempty"`
	BitrateKbps    float64   `json:"bitrateKbps,omitempty"`
	FrameWidth     int       `json:"frameWidth,omitempty"`
	FrameHeight    int       `json:"frameHeight,omitempty"`
	FPS            float64   `json:"fps,omitempty"`
	FramesDecoded  int       `json:"framesDecoded,omitempty"`
	TotalFramesLost int      `json:"totalFramesLost,omitempty"`
	CandidateType  string    `json:"candidateType,omitempty"`
	SnapshotAt     time.Time `json:"snapshotAt"`
}
