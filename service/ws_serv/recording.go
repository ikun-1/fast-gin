package ws_serv

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fast-gin/global"
	"fast-gin/models"

	"github.com/at-wat/ebml-go/webm"
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
// WebM container writer (VP8 video + Opus audio in one file)
// ---------------------------------------------------------------------------

// webmRecorderWriter records VP8 video and Opus audio into a single WebM file.
// It lazily initializes the muxer once both tracks are configured.
type webmRecorderWriter struct {
	file     *os.File
	filePath string

	mu        sync.Mutex
	closeRef  int  // close reference count
	muxerInit bool // muxer has been initialized
	closed    bool

	// WebM track writers (set during initMuxer)
	videoWriter webm.BlockWriteCloser
	audioWriter webm.BlockWriteCloser
	hasVideo    bool
	hasAudio    bool

	// VP8 depacketization state (video)
	vp8Buf            []byte
	vp8Timestamp      uint64
	vp8Started        bool // first S=1 received
	vp8LastSeq        uint16
	vp8SeqInited      bool
	firstKeyFrameSeen bool // don't write non-key frames until one arrives

	// Timing
	videoBaseTS int64 // RTP timestamp of first video frame, for relativization
	audioBaseTS int64 // RTP timestamp of first audio frame

	// Pending — buffered writes before muxer is ready
	pending []func()
}

func newWebMWriter(path string) (*webmRecorderWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create webm file: %w", err)
	}
	return &webmRecorderWriter{
		file:     f,
		filePath: path,
	}, nil
}

// addTrack informs the writer that a track of the given kind exists.
// On first call after both tracks are known, initMuxer is triggered.
func (w *webmRecorderWriter) addTrack(kind webrtc.RTPCodecType) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if kind == webrtc.RTPCodecTypeVideo {
		w.hasVideo = true
	} else {
		w.hasAudio = true
	}
	if w.hasVideo && w.hasAudio && !w.muxerInit {
		w.initMuxerLocked()
	}
}

func (w *webmRecorderWriter) initMuxerLocked() {
	if w.muxerInit {
		return
	}

	tracks := make([]webm.TrackEntry, 0, 2)
	if w.hasVideo {
		tracks = append(tracks, webm.TrackEntry{
			TrackNumber: uint64(len(tracks) + 1),
			TrackUID:    uint64(len(tracks) + 1),
			CodecID:     "V_VP8",
			TrackType:   1, // video
			Video:       &webm.Video{},
		})
	}
	if w.hasAudio {
		tracks = append(tracks, webm.TrackEntry{
			TrackNumber: uint64(len(tracks) + 1),
			TrackUID:    uint64(len(tracks) + 1),
			CodecID:     "A_OPUS",
			TrackType:   2, // audio
			Audio: &webm.Audio{
				SamplingFrequency: 48000,
				Channels:          2,
			},
		})
	}

	ws, err := webm.NewSimpleBlockWriter(w.file, tracks)
	if err != nil {
		zap.S().Errorf("WebM muxer init failed: %s", err)
		return
	}

	idx := 0
	if w.hasVideo {
		w.videoWriter = ws[idx]
		idx++
	}
	if w.hasAudio {
		w.audioWriter = ws[idx]
		idx++
	}

	w.muxerInit = true

	// Flush pending writes
	for _, fn := range w.pending {
		fn()
	}
	w.pending = nil
}

func (w *webmRecorderWriter) writeVideo(pkt *rtp.Packet) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return
	}

	// Detect RTP packet discontinuity. If packets are lost/reordered,
	// the current reconstructed VP8 frame is unsafe; drop it and wait
	// for the next key frame to recover decoder state.
	if w.vp8SeqInited {
		expected := w.vp8LastSeq + 1
		if pkt.SequenceNumber != expected {
			w.vp8Buf = w.vp8Buf[:0]
			w.vp8Started = false
			w.firstKeyFrameSeen = false
		}
	}
	w.vp8LastSeq = pkt.SequenceNumber
	w.vp8SeqInited = true

	data, isStart := depacketizeVP8(pkt.Payload)
	if data == nil {
		w.vp8Buf = w.vp8Buf[:0]
		w.vp8Started = false
		w.firstKeyFrameSeen = false
		return
	}

	// Drop all frames until we see a key frame (VP8 header bit 0 = 0).
	if !w.firstKeyFrameSeen {
		if !isStart {
			return
		}
		if len(data) > 0 && data[0]&0x01 == 0 {
			w.firstKeyFrameSeen = true
		} else {
			return
		}
	}

	if !w.vp8Started {
		if !isStart {
			return
		}
		w.vp8Started = true
	}

	if isStart && len(w.vp8Buf) > 0 {
		w.flushVideoFrameLocked()
	}

	if isStart {
		w.vp8Timestamp = uint64(pkt.Header.Timestamp)
		w.vp8Buf = append(w.vp8Buf[:0], data...)
	} else {
		w.vp8Buf = append(w.vp8Buf, data...)
	}

	if pkt.Marker && len(w.vp8Buf) > 0 {
		w.flushVideoFrameLocked()
	}
}

func (w *webmRecorderWriter) flushVideoFrameLocked() {
	if len(w.vp8Buf) == 0 {
		return
	}

	ts := int64(w.vp8Timestamp)
	if w.videoBaseTS == 0 {
		w.videoBaseTS = ts
	}
	tsMs := (ts - w.videoBaseTS) * 1000 / 90000

	isKey := w.vp8Buf[0]&0x01 == 0

	if !w.muxerInit {
		// Buffer until muxer is ready
		buf := make([]byte, len(w.vp8Buf))
		copy(buf, w.vp8Buf)
		w.pending = append(w.pending, func() {
			if _, err := w.videoWriter.Write(isKey, tsMs, buf); err != nil {
				zap.S().Warnf("WebM video write failed: %s", err)
			}
		})
		w.vp8Buf = w.vp8Buf[:0]
		return
	}

	if _, err := w.videoWriter.Write(isKey, tsMs, w.vp8Buf); err != nil {
		zap.S().Warnf("WebM video write failed: %s", err)
	}
	w.vp8Buf = w.vp8Buf[:0]
}

func (w *webmRecorderWriter) writeAudio(pkt *rtp.Packet) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return
	}

	ts := int64(pkt.Header.Timestamp)
	if w.audioBaseTS == 0 {
		w.audioBaseTS = ts
	}
	tsMs := (ts - w.audioBaseTS) * 1000 / 48000

	if !w.muxerInit {
		buf := make([]byte, len(pkt.Payload))
		copy(buf, pkt.Payload)
		w.pending = append(w.pending, func() {
			if _, err := w.audioWriter.Write(true, tsMs, buf); err != nil {
				zap.S().Warnf("WebM audio write failed: %s", err)
			}
		})
		return
	}

	if _, err := w.audioWriter.Write(true, tsMs, pkt.Payload); err != nil {
		zap.S().Warnf("WebM audio write failed: %s", err)
	}
}

// closeRefCount is called by each track when it closes. When all tracks
// have closed, the muxer and file are finalized.
func (w *webmRecorderWriter) closeRefCount() {
	w.mu.Lock()
	w.closeRef++
	if w.closeRef < 2 {
		w.mu.Unlock()
		return
	}
	w.closed = true
	w.mu.Unlock()

	// Flush any remaining VP8 frame
	w.mu.Lock()
	if len(w.vp8Buf) > 0 {
		w.flushVideoFrameLocked()
	}
	w.mu.Unlock()

	// Init muxer with whatever tracks we have
	w.mu.Lock()
	if !w.muxerInit {
		w.initMuxerLocked()
	}
	w.mu.Unlock()

	if w.videoWriter != nil {
		_ = w.videoWriter.Close()
	}
	if w.audioWriter != nil {
		_ = w.audioWriter.Close()
	}
	// The file is closed by the muxer automatically
}

// webmVideoWriter implements RecorderWriter for the video track.
type webmVideoWriter struct {
	inner *webmRecorderWriter
}

func (w *webmVideoWriter) WriteRTP(pkt *rtp.Packet) error {
	w.inner.writeVideo(pkt)
	return nil
}

func (w *webmVideoWriter) Close() error {
	w.inner.closeRefCount()
	return nil
}

// webmAudioWriter implements RecorderWriter for the audio track.
type webmAudioWriter struct {
	inner *webmRecorderWriter
}

func (w *webmAudioWriter) WriteRTP(pkt *rtp.Packet) error {
	w.inner.writeAudio(pkt)
	return nil
}

func (w *webmAudioWriter) Close() error {
	w.inner.closeRefCount()
	return nil
}

// depacketizeVP8 strips the VP8 payload descriptor from the RTP payload
// and returns the raw VP8 frame data along with a flag indicating whether
// this packet begins a new frame (S bit).
func depacketizeVP8(payload []byte) (data []byte, isStart bool) {
	vp8 := &codecs.VP8Packet{}
	data, err := vp8.Unmarshal(payload)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	// New frame starts only on partition 0 start packet.
	isStart = vp8.S == 1 && vp8.PID == 0
	return data, isStart
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

	writers    map[string]*TrackFileWriter    // key: clientID_kind
	sharedPool map[string]*webmRecorderWriter // key: clientID
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
		sharedPool:  make(map[string]*webmRecorderWriter),
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

	// Get or create shared WebM writer for this client
	shared, exists := s.sharedPool[clientID]
	if !exists {
		filename := fmt.Sprintf("%d_%s.webm", userID, sanitizeFilename(displayName))
		filePath := filepath.Join(s.StoragePath, filename)
		var err error
		shared, err = newWebMWriter(filePath)
		if err != nil {
			return nil, err
		}
		s.sharedPool[clientID] = shared

		// Create DB record for this WebM file
		global.DB.Create(&models.RecordingFile{
			RecordingID: s.ID,
			ClientID:    clientID,
			UserID:      userID,
			DisplayName: displayName,
			FilePath:    filePath,
			Kind:        "webm",
			Codec:       "vp8_opus",
		})
	}

	// Register this track with the shared writer
	shared.addTrack(kind)

	// Create a track-specific wrapper
	var rw RecorderWriter
	if kind == webrtc.RTPCodecTypeVideo {
		rw = &webmVideoWriter{inner: shared}
	} else {
		rw = &webmAudioWriter{inner: shared}
	}

	key := writerKey(clientID, kind)
	tw := &TrackFileWriter{
		RecorderWriter: rw,
		path:           shared.filePath,
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

	for _, tw := range s.writers {
		if err := tw.Close(); err != nil {
			zap.S().Warnf("Close recording writer failed path=%s: %s", tw.path, err)
		}
	}

	s.remuxAndRefreshFileSizes()

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

func (s *RecordingSession) remuxAndRefreshFileSizes() {
	var files []models.RecordingFile
	if err := global.DB.Where("recording_id = ?", s.ID).Find(&files).Error; err != nil {
		zap.S().Warnf("Load recording files failed recordingID=%d: %s", s.ID, err)
		return
	}

	ffmpegPath := strings.TrimSpace(global.Config.Recording.FFmpegPath)
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	for _, file := range files {
		if file.Kind != "webm" {
			continue
		}

		inputPath := file.FilePath
		tempPath := inputPath + ".remux.tmp.webm"

		cmd := exec.Command(ffmpegPath,
			"-hide_banner", "-loglevel", "error", "-y",
			"-i", inputPath,
			"-map", "0",
			"-c", "copy",
			tempPath,
		)
		if output, err := cmd.CombinedOutput(); err != nil {
			_ = os.Remove(tempPath)
			zap.S().Warnf("WebM remux failed file=%s err=%s output=%s", inputPath, err, strings.TrimSpace(string(output)))
		} else {
			if err := os.Remove(inputPath); err != nil && !os.IsNotExist(err) {
				zap.S().Warnf("Remove old WebM failed file=%s: %s", inputPath, err)
			} else if err := os.Rename(tempPath, inputPath); err != nil {
				_ = os.Remove(tempPath)
				zap.S().Warnf("Replace remuxed WebM failed file=%s: %s", inputPath, err)
			}
		}

		if stat, err := os.Stat(inputPath); err == nil {
			if err := global.DB.Model(&models.RecordingFile{}).Where("id = ?", file.ID).Update("file_size", stat.Size()).Error; err != nil {
				zap.S().Warnf("Update recording file size failed id=%d: %s", file.ID, err)
			}
		}
	}
}

func writerKey(clientID string, kind webrtc.RTPCodecType) string {
	return clientID + "_" + kind.String()
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
