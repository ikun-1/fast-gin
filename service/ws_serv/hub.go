package ws_serv

import (
	"encoding/json"
	"fast-gin/global"
	"fast-gin/models"
	"sync/atomic"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

type Hub struct {
	Clients    map[string]*Client
	Rooms      map[uint]*Room
	Register   chan *Client
	Unregister chan *Client
	clientSeq  uint64
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]*Client),
		Rooms:      make(map[uint]*Room),
		Register:   make(chan *Client, 256),
		Unregister: make(chan *Client, 256),
	}
}

func (h *Hub) NextClientSeq() uint64 {
	return atomic.AddUint64(&h.clientSeq, 1)
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.ClientID] = client
			zap.S().Infof("Client registered: %s (user=%d)", client.ClientID, client.UserID)

			// Start negotiation handler goroutine for this client
			go h.negotiationLoop(client)

		case client := <-h.Unregister:
			if _, ok := h.Clients[client.ClientID]; ok {
				if client.RoomNo != 0 {
					h.leaveRoom(client)
				}
				if client.PC != nil {
					client.PC.Close()
				}
				delete(h.Clients, client.ClientID)
				close(client.Send)
				zap.S().Infof("Client unregistered: %s", client.ClientID)
			}
		}
	}
}

// negotiationLoop handles renegotiation triggered by OnNegotiationNeeded
func (h *Hub) negotiationLoop(client *Client) {
	for range client.negotiateChan {
		client.HandleNegotiation()
	}
}

func (h *Hub) HandleMessage(client *Client, msg *WsClientMessage) {
	switch msg.Type {
	case "join-room":
		h.handleJoinRoom(client, msg)
	case "leave-room":
		h.handleLeaveRoom(client)
	case "offer":
		h.handleOffer(client, msg)
	case "answer":
		h.handleAnswer(client, msg)
	case "ice-candidate":
		h.handleIceCandidate(client, msg)
	case "mute-toggle":
		h.handleMuteToggle(client, msg)
	case "screen-share-start":
		h.handleScreenShareStart(client)
	case "screen-share-stop":
		h.handleScreenShareStop(client)
	case "chat-message":
		h.handleChatMessage(client, msg)
	case "recording-control":
		h.handleRecordingControl(client, msg)
	default:
		client.SendJSON(WsServerMessage{Type: "error", Data: "unknown message type"})
	}
}

func (h *Hub) handleJoinRoom(client *Client, msg *WsClientMessage) {
	roomNo := msg.RoomNo
	if roomNo == 0 {
		client.SendJSON(WsServerMessage{Type: "error", Data: "roomNo is required"})
		return
	}

	// Verify meeting exists
	var meeting models.Meeting
	if err := global.DB.Where("room_no = ?", roomNo).First(&meeting).Error; err != nil {
		client.SendJSON(WsServerMessage{Type: "error", Data: "会议不存在"})
		return
	}

	if meeting.Status == "ended" {
		client.SendJSON(WsServerMessage{Type: "error", Data: "会议已结束"})
		return
	}

	// Check max participants
	room, exists := h.Rooms[roomNo]
	if exists && len(room.Clients) >= global.Config.WebRTC.MaxParticipants {
		client.SendJSON(WsServerMessage{Type: "error", Data: "会议人数已满"})
		return
	}

	client.RoomNo = roomNo
	client.IsHost = meeting.HostID == client.UserID

	// Get or create room
	if !exists {
		room = NewRoom(roomNo)
		h.Rooms[roomNo] = room
	}

	// Add client to room
	room.AddClient(client)

	// Build participant list (include all clients including self)
	participants := make([]ParticipantInfo, 0)
	room.ForEachClient(func(id string, c *Client) {
		participants = append(participants, ParticipantInfo{
			ClientID:    c.ClientID,
			DisplayName: c.DisplayName,
			IsHost:      c.IsHost,
		})
	})

	// Notify the joining client
	client.SendJSON(WsServerMessage{
		Type: "room-joined",
		Data: RoomJoinedData{
			RoomNo:       roomNo,
			ClientID:     client.ClientID,
			Participants: participants,
		},
	})

	// Notify existing participants
	room.Broadcast(WsServerMessage{
		Type: "user-joined",
		Data: UserJoinedData{
			ClientID:    client.ClientID,
			DisplayName: client.DisplayName,
			IsHost:      client.IsHost,
		},
	}, client.ClientID)
}

func (h *Hub) leaveRoom(client *Client) {
	room, exists := h.Rooms[client.RoomNo]
	if !exists {
		return
	}

	room.RemoveClient(client.ClientID)
	client.RoomNo = 0

	// Notify remaining participants
	room.Broadcast(WsServerMessage{
		Type: "user-left",
		Data: UserLeftData{ClientID: client.ClientID},
	}, "")

	// Clean up empty rooms
	room.mu.RLock()
	empty := len(room.Clients) == 0
	room.mu.RUnlock()
	if empty {
		delete(h.Rooms, client.RoomNo)
		zap.S().Infof("Room %d removed (empty)", client.RoomNo)
	}
}

func (h *Hub) handleLeaveRoom(client *Client) {
	if client.RoomNo == 0 {
		return
	}
	h.leaveRoom(client)
}

// ---------- SFU: WebRTC Signaling ----------

func (h *Hub) handleOffer(client *Client, msg *WsClientMessage) {
	var sdp webrtc.SessionDescription
	if err := json.Unmarshal(msg.SDP, &sdp); err != nil {
		zap.S().Errorf("Invalid SDP from client=%s: %s", client.ClientID, err)
		return
	}

	iceServers := make([]webrtc.ICEServer, 0)
	for _, s := range global.Config.WebRTC.ICEServers {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:       s.URLs,
			Username:   s.Username,
			Credential: s.Credential,
		})
	}

	config := webrtc.Configuration{
		ICEServers: iceServers,
	}

	// Create PeerConnection if it doesn't exist yet
	if client.PC == nil {
		pc, err := client.CreatePeerConnection(config)
		if err != nil {
			zap.S().Errorf("Create PC failed client=%s: %s", client.ClientID, err)
			return
		}

		room, exists := h.Rooms[client.RoomNo]
		if !exists {
			zap.S().Errorf("Room not found for client=%s", client.ClientID)
			return
		}

		// Handle incoming tracks from this client
		pc.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			zap.S().Infof("Received track client=%s kind=%s id=%s codec=%s",
				client.ClientID, remoteTrack.Kind(), remoteTrack.ID(), remoteTrack.Codec().MimeType)
			room.RegisterTrack(client.ClientID, remoteTrack, receiver)
		})
	}

	if err := client.PC.SetRemoteDescription(sdp); err != nil {
		zap.S().Errorf("SetRemoteDescription failed client=%s: %s", client.ClientID, err)
		return
	}

	// Create answer
	answer, err := client.PC.CreateAnswer(nil)
	if err != nil {
		zap.S().Errorf("CreateAnswer failed client=%s: %s", client.ClientID, err)
		return
	}

	if err := client.PC.SetLocalDescription(answer); err != nil {
		zap.S().Errorf("SetLocalDescription failed client=%s: %s", client.ClientID, err)
		return
	}

	client.SendJSON(WsServerMessage{
		Type: "answer",
		Data: answer,
	})

	// Subscribe this client to existing tracks from other participants
	// Do this immediately after initial negotiation, not waiting for OnTrack
	if room, exists := h.Rooms[client.RoomNo]; exists {
		room.SubscribeExistingTracks(client)
	}
}

func (h *Hub) handleAnswer(client *Client, msg *WsClientMessage) {
	var sdp webrtc.SessionDescription
	if err := json.Unmarshal(msg.SDP, &sdp); err != nil {
		zap.S().Errorf("Invalid answer SDP from client=%s: %s", client.ClientID, err)
		return
	}

	if client.PC == nil {
		zap.S().Warnf("No PC for answer client=%s", client.ClientID)
		return
	}

	if err := client.PC.SetRemoteDescription(sdp); err != nil {
		zap.S().Errorf("SetRemoteDescription (answer) failed client=%s: %s", client.ClientID, err)
		return
	}

	// After renegotiation completes (answer processed), send PLI for ALL
	// video tracks (including this client's own). This ensures:
	// 1. New subscribers get a keyframe immediately instead of waiting
	//    for the periodic keyframe (which causes black screen).
	// 2. When a client finishes renegotiation after remote tracks were
	//    added to its PC, it triggers a keyframe for its own outgoing
	//    video that other mid-renegotiation subscribers may have missed.
	if client.RoomNo != 0 {
		room, exists := h.Rooms[client.RoomNo]
		if !exists {
			return
		}
		room.trackMu.RLock()
		for srcID, tracks := range room.TrackLocals {
			for _, info := range tracks {
				if info.RemoteTrack.Kind() == webrtc.RTPCodecTypeVideo {
					room.mu.RLock()
					srcClient, ok := room.Clients[srcID]
					room.mu.RUnlock()
					if ok && srcClient.PC != nil {
						go func(ssrc webrtc.SSRC, pc *webrtc.PeerConnection) {
							if err := pc.WriteRTCP([]rtcp.Packet{
								&rtcp.PictureLossIndication{MediaSSRC: uint32(ssrc)},
							}); err != nil {
								zap.S().Warnf("PLI write after answer failed client=%s: %s", client.ClientID, err)
							}
						}(info.RemoteTrack.SSRC(), srcClient.PC)
					}
				}
			}
		}
		room.trackMu.RUnlock()
	}
}

func (h *Hub) handleIceCandidate(client *Client, msg *WsClientMessage) {
	if client.PC == nil {
		return
	}

	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal(msg.Candidate, &candidate); err != nil {
		zap.S().Warnf("Invalid ICE candidate from client=%s: %s", client.ClientID, err)
		return
	}

	if err := client.PC.AddICECandidate(candidate); err != nil {
		zap.S().Warnf("AddICECandidate failed client=%s: %s", client.ClientID, err)
	}
}

// ---------- Room Events ----------

func (h *Hub) handleMuteToggle(client *Client, msg *WsClientMessage) {
	room, exists := h.Rooms[client.RoomNo]
	if !exists {
		return
	}
	room.Broadcast(WsServerMessage{
		Type: "user-muted",
		Data: MuteToggleData{
			ClientID: client.ClientID,
			Muted:    msg.Muted,
			Kind:     msg.Kind,
		},
	}, "")
}

func (h *Hub) handleScreenShareStart(client *Client) {
	room, exists := h.Rooms[client.RoomNo]
	if !exists {
		return
	}
	room.Broadcast(WsServerMessage{
		Type: "screen-share-started",
		Data: ScreenShareData{ClientID: client.ClientID},
	}, "")
}

func (h *Hub) handleScreenShareStop(client *Client) {
	room, exists := h.Rooms[client.RoomNo]
	if !exists {
		return
	}
	room.Broadcast(WsServerMessage{
		Type: "screen-share-stopped",
		Data: ScreenShareData{ClientID: client.ClientID},
	}, "")
}

func (h *Hub) handleChatMessage(client *Client, msg *WsClientMessage) {
	room, exists := h.Rooms[client.RoomNo]
	if !exists {
		return
	}
	room.Broadcast(WsServerMessage{
		Type: "chat-message",
		Data: ChatMessageData{
			FromClientID: client.ClientID,
			DisplayName:  client.DisplayName,
			Text:         msg.Text,
		},
	}, "")
}

func (h *Hub) handleRecordingControl(client *Client, msg *WsClientMessage) {
	room, exists := h.Rooms[client.RoomNo]
	if !exists {
		return
	}

	// Only the host can control recording
	if !client.IsHost {
		client.SendJSON(WsServerMessage{Type: "error", Data: "仅主持人可以控制录制"})
		return
	}

	switch msg.Action {
	case "start":
		if GlobalRecordingManager.IsRecording(client.RoomNo) {
			client.SendJSON(WsServerMessage{Type: "error", Data: "录制已在进行中"})
			return
		}
		var meeting models.Meeting
		if err := global.DB.Where("room_no = ?", client.RoomNo).First(&meeting).Error; err != nil {
			client.SendJSON(WsServerMessage{Type: "error", Data: "会议不存在"})
			return
		}
		session, err := GlobalRecordingManager.StartSession(client.RoomNo, meeting.ID, client.UserID)
		if err != nil {
			client.SendJSON(WsServerMessage{Type: "error", Data: "启动录制失败: " + err.Error()})
			return
		}
		room.trackMu.RLock()
		for clientID, tracks := range room.TrackLocals {
			for _, info := range tracks {
				if c := room.GetClient(clientID); c != nil {
					if _, err := session.EnsureWriter(clientID, c.UserID, c.DisplayName, info.RemoteTrack.Codec().MimeType); err != nil {
						zap.S().Warnf("Recording EnsureWriter failed: %s", err)
					}
				}
			}
		}
		// Request key frames from all video sources so the recording
		// contains a key frame to start decoding from
		for srcID, tracks := range room.TrackLocals {
			for _, info := range tracks {
				if info.RemoteTrack.Kind() == webrtc.RTPCodecTypeVideo {
					room.mu.RLock()
					srcClient, ok := room.Clients[srcID]
					room.mu.RUnlock()
					if ok && srcClient.PC != nil {
						go func(ssrc webrtc.SSRC, pc *webrtc.PeerConnection) {
							if err := pc.WriteRTCP([]rtcp.Packet{
								&rtcp.PictureLossIndication{MediaSSRC: uint32(ssrc)},
							}); err != nil {
								zap.S().Warnf("Recording PLI failed client=%s: %s", srcID, err)
							}
						}(info.RemoteTrack.SSRC(), srcClient.PC)
					}
				}
			}
		}
		room.trackMu.RUnlock()
		room.SetRecorder(session)
		room.Broadcast(WsServerMessage{
			Type: "recording-started",
			Data: RecordingControlData{
				Action:    "started",
				StartedAt: session.StartedAt.Format("2006-01-02 15:04:05"),
			},
		}, "")
	case "stop":
		if !GlobalRecordingManager.IsRecording(client.RoomNo) {
			client.SendJSON(WsServerMessage{Type: "error", Data: "当前没有进行中的录制"})
			return
		}
		room.ClearRecorder()
		durationMs, err := GlobalRecordingManager.StopSession(client.RoomNo)
		if err != nil {
			client.SendJSON(WsServerMessage{Type: "error", Data: "停止录制失败: " + err.Error()})
			return
		}
		room.Broadcast(WsServerMessage{
			Type: "recording-stopped",
			Data: RecordingControlData{
				Action:     "stopped",
				DurationMs: durationMs,
			},
		}, "")
	}
}
