package ws_serv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fast-gin/global"
	"fast-gin/models"

	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// RecorderWriter defines the interface for writing recorded media.
type RecorderWriter interface {
	WriteRTP(pkt *rtp.Packet) error
	Close() error
}

// ---------------------------------------------------------------------------
// TrackFileWriter
// ---------------------------------------------------------------------------

type TrackFileWriter struct {
	RecorderWriter
	path     string
	clientID string
	kind     webrtc.RTPCodecType
}

// ---------------------------------------------------------------------------
// RecordingSession
// ---------------------------------------------------------------------------

type RecordingSession struct {
	ID          uint
	RoomNo      uint
	MeetingID   uint
	HostID      uint
	StartedAt   time.Time
	StoragePath string

	writers    map[string]*TrackFileWriter // key: clientID_kind
	sharedPool map[string]*ClientRecorder  // key: clientID
	writersMu  sync.RWMutex
	closed     bool
}

func NewRecordingSession(roomNo, meetingID, hostID uint) (*RecordingSession, error) {
	now := time.Now()
	storageDir := filepath.Join(global.Config.Recording.Dir, fmt.Sprintf("%d", roomNo), fmt.Sprintf("%d", now.Unix()))

	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("create recording dir: %w", err)
	}

	session := &RecordingSession{
		RoomNo:      roomNo,
		MeetingID:   meetingID,
		HostID:      hostID,
		StartedAt:   now,
		StoragePath: storageDir,
		writers:     make(map[string]*TrackFileWriter),
		sharedPool:  make(map[string]*ClientRecorder),
	}

	rec := &models.Recording{
		MeetingID:   meetingID,
		RoomNo:      roomNo,
		HostID:      hostID,
		StartedAt:   now,
		Status:      "recording",
		StoragePath: storageDir,
	}
	if err := global.DB.Create(rec).Error; err != nil {
		os.RemoveAll(storageDir)
		return nil, fmt.Errorf("create recording db record: %w", err)
	}
	session.ID = rec.ID

	zap.S().Infof("Recording session started: id=%d room=%d path=%s", session.ID, roomNo, storageDir)
	return session, nil
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}

func (s *RecordingSession) createWriter(clientID string, userID uint, displayName string, mimeType string) (*TrackFileWriter, error) {
	kind := webrtc.RTPCodecTypeVideo
	if strings.HasPrefix(strings.ToLower(mimeType), "audio/") {
		kind = webrtc.RTPCodecTypeAudio
	}

	s.writersMu.Lock()
	defer s.writersMu.Unlock()

	// Get or create ClientRecorder for this participant
	cr, exists := s.sharedPool[clientID]
	if !exists {
		baseName := fmt.Sprintf("%d_%s", userID, sanitizeFilename(displayName))
		videoPath := filepath.Join(s.StoragePath, "."+baseName+".ivf")
		audioPath := filepath.Join(s.StoragePath, "."+baseName+".opus.raw")
		outputPath := filepath.Join(s.StoragePath, baseName+".webm")

		videoWriter, err := NewIVFWriter(videoPath)
		if err != nil {
			return nil, fmt.Errorf("create ivf writer: %w", err)
		}
		audioWriter, err := NewTempOpusWriter(audioPath)
		if err != nil {
			videoWriter.Close()
			return nil, fmt.Errorf("create opus writer: %w", err)
		}

		cr = &ClientRecorder{
			clientID:    clientID,
			videoWriter: videoWriter,
			audioWriter: audioWriter,
			videoPath:   videoPath,
			audioPath:   audioPath,
			outputPath:  outputPath,
			hasVideo:    false,
			hasAudio:    false,
		}
		s.sharedPool[clientID] = cr

		// Create DB record for the final WebM file
		global.DB.Create(&models.RecordingFile{
			RecordingID: s.ID,
			ClientID:    clientID,
			UserID:      userID,
			DisplayName: displayName,
			FilePath:    outputPath,
			Kind:        "webm",
			Codec:       "vp8_opus",
		})
	}

	// Mark track presence and create track-specific wrapper
	if kind == webrtc.RTPCodecTypeVideo {
		cr.hasVideo = true
	} else {
		cr.hasAudio = true
	}

	var rw RecorderWriter
	path := cr.outputPath
	if kind == webrtc.RTPCodecTypeVideo {
		rw = cr.videoWriter
	} else {
		rw = cr.audioWriter
	}

	key := writerKey(clientID, kind)
	tw := &TrackFileWriter{
		RecorderWriter: rw,
		path:           path,
		clientID:       clientID,
		kind:           kind,
	}
	s.writers[key] = tw

	return tw, nil
}

func (s *RecordingSession) EnsureWriter(clientID string, userID uint, displayName string, mimeType string) (*TrackFileWriter, error) {
	kind := webrtc.RTPCodecTypeVideo
	if strings.HasPrefix(strings.ToLower(mimeType), "audio/") {
		kind = webrtc.RTPCodecTypeAudio
	}

	s.writersMu.RLock()
	tw, exists := s.writers[writerKey(clientID, kind)]
	s.writersMu.RUnlock()
	if exists {
		return tw, nil
	}

	return s.createWriter(clientID, userID, displayName, mimeType)
}

func (s *RecordingSession) GetWriter(clientID string, kind webrtc.RTPCodecType) *TrackFileWriter {
	key := writerKey(clientID, kind)
	s.writersMu.RLock()
	defer s.writersMu.RUnlock()
	return s.writers[key]
}

func (s *RecordingSession) Stop() (int64, error) {
	s.writersMu.Lock()
	defer s.writersMu.Unlock()

	if s.closed {
		return 0, errors.New("session already closed")
	}
	s.closed = true

	durationMs := time.Since(s.StartedAt).Milliseconds()
	totalFiles := len(s.sharedPool)

	// Close all track writers (flushes IVF + temp audio)
	for _, tw := range s.writers {
		if err := tw.Close(); err != nil {
			zap.S().Warnf("Close recording writer failed path=%s: %s", tw.path, err)
		}
	}

	// Remux each client's temp files into final WebM via ffmpeg
	ffmpegPath := strings.TrimSpace(global.Config.Recording.FFmpegPath)
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	for _, cr := range s.sharedPool {
		if !cr.hasVideo && !cr.hasAudio {
			continue
		}
		if err := cr.Remux(ffmpegPath); err != nil {
			zap.S().Warnf("Client remux failed client=%s: %s", cr.clientID, err)
			continue
		}

		// Update file size in DB
		if stat, err := os.Stat(cr.outputPath); err == nil {
			global.DB.Model(&models.RecordingFile{}).
				Where("recording_id = ? AND client_id = ?", s.ID, cr.clientID).
				Update("file_size", stat.Size())
		}
	}

	now := time.Now()
	updates := map[string]any{
		"ended_at":    now,
		"duration_ms": durationMs,
		"status":      "completed",
		"file_count":  totalFiles,
	}
	if err := global.DB.Model(&models.Recording{}).Where("id = ?", s.ID).Updates(updates).Error; err != nil {
		zap.S().Errorf("Update recording db record failed id=%d: %s", s.ID, err)
	}

	zap.S().Infof("Recording session stopped: id=%d duration=%dms files=%d",
		s.ID, durationMs, totalFiles)

	return durationMs, nil
}

func writerKey(clientID string, kind webrtc.RTPCodecType) string {
	return clientID + "_" + kind.String()
}

// ---------------------------------------------------------------------------
// depacketizeVP8
// ---------------------------------------------------------------------------

func depacketizeVP8(payload []byte) (data []byte, isStart bool) {
	vp8 := &codecs.VP8Packet{}
	data, err := vp8.Unmarshal(payload)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	isStart = vp8.S == 1 && vp8.PID == 0
	return data, isStart
}

// ---------------------------------------------------------------------------
// RecordingManager singleton
// ---------------------------------------------------------------------------

type RecordingManager struct {
	active map[uint]*RecordingSession
	mu     sync.RWMutex
}

var GlobalRecordingManager = &RecordingManager{
	active: make(map[uint]*RecordingSession),
}

func (m *RecordingManager) GetSession(roomNo uint) *RecordingSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active[roomNo]
}

func (m *RecordingManager) IsRecording(roomNo uint) bool {
	return m.GetSession(roomNo) != nil
}

func (m *RecordingManager) StartSession(roomNo, meetingID, hostID uint) (*RecordingSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.active[roomNo]; exists {
		return nil, errors.New("recording already active for this room")
	}

	session, err := NewRecordingSession(roomNo, meetingID, hostID)
	if err != nil {
		return nil, err
	}

	m.active[roomNo] = session
	return session, nil
}

func (m *RecordingManager) StopSession(roomNo uint) (int64, error) {
	m.mu.Lock()
	session, exists := m.active[roomNo]
	if !exists {
		m.mu.Unlock()
		return 0, errors.New("no active recording for this room")
	}
	delete(m.active, roomNo)
	m.mu.Unlock()

	return session.Stop()
}
