package ws_serv

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

type TrackInfo struct {
	RemoteTrack *webrtc.TrackRemote
	LocalTrack  *webrtc.TrackLocalStaticRTP
	ClientID    string
	StopFn      func()
}

type Room struct {
	RoomNo  uint
	Clients map[string]*Client
	mu      sync.RWMutex

	TrackLocals map[string][]*TrackInfo
	trackMu     sync.RWMutex

	recorder atomic.Pointer[RecordingSession]
}

func NewRoom(roomNo uint) *Room {
	return &Room{
		RoomNo:      roomNo,
		Clients:     make(map[string]*Client),
		TrackLocals: make(map[string][]*TrackInfo),
	}
}

func (r *Room) AddClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Clients[client.ClientID] = client
}

func (r *Room) RemoveClient(clientID string) {
	r.mu.Lock()
	client, exists := r.Clients[clientID]
	if exists {
		if client.PC != nil {
			client.PC.Close()
		}
	}
	delete(r.Clients, clientID)
	r.mu.Unlock()

	// Stop track relay goroutines
	r.trackMu.Lock()
	for _, info := range r.TrackLocals[clientID] {
		if info.StopFn != nil {
			info.StopFn()
		}
	}
	delete(r.TrackLocals, clientID)
	r.trackMu.Unlock()

	// Clean up local tracks on other clients
	r.mu.RLock()
	for _, c := range r.Clients {
		updated := make([]*webrtc.TrackLocalStaticRTP, 0)
		for _, lt := range c.LocalTracks {
			if lt != nil {
				updated = append(updated, lt)
			}
		}
		c.LocalTracks = updated
	}
	r.mu.RUnlock()
}

func (r *Room) GetClient(clientID string) *Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Clients[clientID]
}

func (r *Room) Broadcast(msg interface{}, excludeClientID string) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for clientID, client := range r.Clients {
		if clientID == excludeClientID {
			continue
		}
		select {
		case client.Send <- data:
		default:
		}
	}
}

func (r *Room) ForEachClient(fn func(clientID string, client *Client)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for clientID, client := range r.Clients {
		fn(clientID, client)
	}
}

func (r *Room) SetRecorder(s *RecordingSession) {
	r.recorder.Store(s)
}

func (r *Room) GetRecorder() *RecordingSession {
	return r.recorder.Load()
}

func (r *Room) ClearRecorder() {
	r.recorder.Store(nil)
}

// ---------- SFU: Track Management ----------

func (r *Room) RegisterTrack(clientID string, remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	// Register track info under write lock
	r.trackMu.Lock()
	localTrack, err := webrtc.NewTrackLocalStaticRTP(
		remoteTrack.Codec().RTPCodecCapability,
		fmt.Sprintf("track_%s_%s", clientID, remoteTrack.ID()),
		fmt.Sprintf("stream_%s", clientID),
	)
	if err != nil {
		r.trackMu.Unlock()
		zap.S().Errorf("Create local track failed client=%s: %s", clientID, err)
		return
	}

	stopCh := make(chan struct{})
	var stopOnce sync.Once

	info := &TrackInfo{
		RemoteTrack: remoteTrack,
		LocalTrack:  localTrack,
		ClientID:    clientID,
		StopFn:      func() { stopOnce.Do(func() { close(stopCh) }) },
	}
	r.TrackLocals[clientID] = append(r.TrackLocals[clientID], info)
	r.trackMu.Unlock()

	// Relay: read RTP from remote track, write to local track + recording
	go func() {
		buf := make([]byte, 1460)
		kind := remoteTrack.Kind()
		for {
			select {
			case <-stopCh:
				return
			default:
				n, _, readErr := remoteTrack.Read(buf)
				if readErr != nil {
					return
				}
				var pkt rtp.Packet
				if err := pkt.Unmarshal(buf[:n]); err != nil {
					continue
				}
				if writeErr := localTrack.WriteRTP(&pkt); writeErr != nil {
					return
				}
				// Best-effort recording capture (never blocks the relay)
				if recorder := r.GetRecorder(); recorder != nil {
					if tw := recorder.GetWriter(clientID, kind); tw != nil {
						if err := tw.WriteRTP(&pkt); err != nil {
							zap.S().Warnf("Recording write failed client=%s: %s", clientID, err)
						}
					}
				}
			}
		}
	}()

	// If recording is active, ensure this track has a writer
	if recorder := r.GetRecorder(); recorder != nil {
		if c := r.GetClient(clientID); c != nil {
			if _, err := recorder.EnsureWriter(clientID, c.UserID, c.DisplayName, remoteTrack.Codec().MimeType); err != nil {
				zap.S().Warnf("Failed to create recording writer for client=%s: %s", clientID, err)
			}
		}
	}

	// Subscribe this new track to all other clients
	r.mu.RLock()
	for id, client := range r.Clients {
		if id == clientID {
			continue
		}
		r.addTrackToPeer(client, localTrack, uint32(remoteTrack.SSRC()))
	}
	r.mu.RUnlock()

	// Request keyframe via PLI so new subscribers see video immediately
	if remoteTrack.Kind() == webrtc.RTPCodecTypeVideo {
		r.mu.RLock()
		sourceClient, ok := r.Clients[clientID]
		r.mu.RUnlock()
		if ok && sourceClient.PC != nil {
			go func(ssrc webrtc.SSRC, pc *webrtc.PeerConnection) {
				if err := pc.WriteRTCP([]rtcp.Packet{
					&rtcp.PictureLossIndication{MediaSSRC: uint32(ssrc)},
				}); err != nil {
					zap.S().Warnf("PLI write failed client=%s: %s", clientID, err)
				}
			}(remoteTrack.SSRC(), sourceClient.PC)
		}
	}
}

func (r *Room) addTrackToPeer(client *Client, track *webrtc.TrackLocalStaticRTP, sourceSSRC uint32) {
	if client.PC == nil {
		return
	}

	// Dedup: skip if this track is already added
	for _, t := range client.LocalTracks {
		if t == track {
			return
		}
	}

	sender, err := client.PC.AddTrack(track)
	if err != nil {
		zap.S().Errorf("AddTrack failed client=%s: %s", client.ClientID, err)
		return
	}

	// Extract source client ID from the relay track's stream ID ("stream_{clientID}")
	srcClientID := strings.TrimPrefix(track.StreamID(), "stream_")

	// Read RTCP from the subscriber and forward PLI/FIR keyframe requests
	// to the source client. The subscriber's PLI has the relay track's SSRC
	// (assigned by Pion), but the source browser only recognizes its own
	// outgoing SSRC — so we rewrite MediaSSRC before forwarding.
	go func() {
		for {
			packets, _, rtcpErr := sender.ReadRTCP()
			if rtcpErr != nil {
				return
			}
			for _, pkt := range packets {
				switch p := pkt.(type) {
				case *rtcp.PictureLossIndication:
					if srcClientID == "" || srcClientID == client.ClientID {
						continue
					}
					// Rewrite SSRC to match the source's original SSRC;
					// without this the source browser ignores the PLI.
					p.MediaSSRC = sourceSSRC
					r.mu.RLock()
					srcClient, ok := r.Clients[srcClientID]
					r.mu.RUnlock()
					if ok && srcClient.PC != nil {
						if err := srcClient.PC.WriteRTCP([]rtcp.Packet{p}); err != nil {
							zap.S().Warnf("RTCP PLI forward failed: %s", err)
						}
					}
				case *rtcp.FullIntraRequest:
					if srcClientID == "" || srcClientID == client.ClientID {
						continue
					}
					p.MediaSSRC = sourceSSRC
					r.mu.RLock()
					srcClient, ok := r.Clients[srcClientID]
					r.mu.RUnlock()
					if ok && srcClient.PC != nil {
						if err := srcClient.PC.WriteRTCP([]rtcp.Packet{p}); err != nil {
							zap.S().Warnf("RTCP FIR forward failed: %s", err)
						}
					}
				}
			}
		}
	}()

	client.LocalTracks = append(client.LocalTracks, track)
}

func (r *Room) SubscribeExistingTracks(newClient *Client) {
	r.trackMu.RLock()

	zap.S().Infof("SubscribeExistingTracks for client=%s, existing tracks=%d",
		newClient.ClientID, len(r.TrackLocals))

	type pliReq struct {
		ssrc webrtc.SSRC
		pc   *webrtc.PeerConnection
	}
	var pliReqs []pliReq

	for clientID, tracks := range r.TrackLocals {
		if clientID == newClient.ClientID {
			continue
		}
		for _, info := range tracks {
			r.addTrackToPeer(newClient, info.LocalTrack, uint32(info.RemoteTrack.SSRC()))

			// If recording is active, ensure writers for existing tracks
			if recorder := r.GetRecorder(); recorder != nil {
				if c := r.GetClient(clientID); c != nil {
					if _, err := recorder.EnsureWriter(clientID, c.UserID, c.DisplayName, info.RemoteTrack.Codec().MimeType); err != nil {
						zap.S().Warnf("Failed to create recording writer for client=%s: %s", clientID, err)
					}
				}
			}

			// Collect PLI requests: ask the source to send a keyframe
			// so the new subscriber can decode immediately instead of
			// waiting for the next periodic keyframe (which causes black screen).
			if info.RemoteTrack.Kind() == webrtc.RTPCodecTypeVideo {
				r.mu.RLock()
				srcClient, ok := r.Clients[clientID]
				r.mu.RUnlock()
				if ok && srcClient.PC != nil {
					pliReqs = append(pliReqs, pliReq{
						ssrc: info.RemoteTrack.SSRC(),
						pc:   srcClient.PC,
					})
				}
			}
		}
	}
	r.trackMu.RUnlock()

	// Send PLI requests after releasing locks
	for _, req := range pliReqs {
		zap.S().Infof("Sending PLI for subscriber client=%s ssrc=%d",
			newClient.ClientID, uint32(req.ssrc))
		go func(ssrc webrtc.SSRC, pc *webrtc.PeerConnection) {
			if err := pc.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{MediaSSRC: uint32(ssrc)},
			}); err != nil {
				zap.S().Warnf("PLI write for subscriber failed: %s", err)
			}
		}(req.ssrc, req.pc)
	}
}
