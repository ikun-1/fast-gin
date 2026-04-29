package ws_serv

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/pion/rtp"
	"go.uber.org/zap"
)

// IVFRecorderWriter records VP8 video in IVF format.
// IVF is a simple container that ffmpeg reads natively:
//
//	File header (32 bytes): magic "DKIF", codec "VP80", width, height, timebase
//	Per frame  (12+ bytes): 12-byte frame header + raw VP8 frame data
//
// The .ivf file is a temp intermediate; the final .webm is produced by ffmpeg at Stop().
type IVFRecorderWriter struct {
	file     *os.File
	filePath string
	closed   bool

	mu             sync.Mutex
	wroteHeader    bool
	frameCount     int
	frameTimestamp uint64 // IVF frame timestamp (ms), to guarantee monotonicity

	// VP8 depacketization state
	vp8Buf            []byte
	vp8Timestamp      uint64
	vp8Started        bool
	vp8LastSeq        uint16
	vp8SeqInited      bool
	firstKeyFrameSeen bool

	// RTP timestamp relativization
	videoBaseTS  int64
	lastFlushedTS int64

	// Resolution (parsed from first key frame)
	width  uint16
	height uint16
}

func NewIVFWriter(path string) (*IVFRecorderWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create ivf file: %w", err)
	}
	return &IVFRecorderWriter{file: f, filePath: path}, nil
}

// WriteRTP implements RecorderWriter. It depacketizes VP8 from RTP,
// reassembles frames, and writes them in IVF format.
func (w *IVFRecorderWriter) WriteRTP(pkt *rtp.Packet) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	// Detect RTP packet discontinuity (loss or reorder)
	if w.vp8SeqInited {
		expected := w.vp8LastSeq + 1
		if pkt.SequenceNumber != expected {
			w.vp8Buf = w.vp8Buf[:0]
			w.vp8Started = false
		}
	}
	w.vp8LastSeq = pkt.SequenceNumber
	w.vp8SeqInited = true

	data, isStart := depacketizeVP8(pkt.Payload)
	if data == nil {
		w.vp8Buf = w.vp8Buf[:0]
		w.vp8Started = false
		w.firstKeyFrameSeen = false
		return nil
	}

	// Drop everything until we see a key frame
	if !w.firstKeyFrameSeen {
		if !isStart {
			return nil
		}
		if len(data) > 0 && data[0]&0x01 == 0 {
			w.firstKeyFrameSeen = true
			// Parse resolution from first key frame
			if vw, vh, ok := parseVP8Resolution(data); ok {
				w.width = vw
				w.height = vh
			}
		} else {
			return nil
		}
	}

	if !w.vp8Started {
		if !isStart {
			return nil
		}
		w.vp8Started = true
	}

	if isStart && len(w.vp8Buf) > 0 {
		w.flushFrame()
	}

	if isStart {
		w.vp8Timestamp = uint64(pkt.Header.Timestamp)
		w.vp8Buf = append(w.vp8Buf[:0], data...)
	} else {
		w.vp8Buf = append(w.vp8Buf, data...)
	}

	if pkt.Marker && len(w.vp8Buf) > 0 {
		w.flushFrame()
	}
	return nil
}

func (w *IVFRecorderWriter) flushFrame() {
	if len(w.vp8Buf) == 0 {
		return
	}

	// Compute monotonic timestamp in milliseconds
	ts := int64(w.vp8Timestamp)
	if w.videoBaseTS == 0 {
		w.videoBaseTS = ts
	}
	tsMs := (ts - w.videoBaseTS) * 1000 / 90000

	// If this is the first frame ever, write IVF header first
	if !w.wroteHeader {
		w.writeHeader()
	}
	w.wroteHeader = true

	// Clamp to last flushed timestamp to guarantee IVF frame timestamp monotonicity
	if tsMs <= w.lastFlushedTS {
		tsMs = w.lastFlushedTS + 1
	}
	w.lastFlushedTS = tsMs

	// Use the IVF monotonic counter as the timestamp
	w.frameCount++
	w.frameTimestamp = uint64(tsMs)

	buf := make([]byte, len(w.vp8Buf))
	copy(buf, w.vp8Buf)
	w.vp8Buf = w.vp8Buf[:0]

	w.writeIVFFrame(buf, w.frameTimestamp)
}

func (w *IVFRecorderWriter) writeHeader() {
	// Use sensible defaults if resolution wasn't parsed from VP8 bitstream.
	// The VP8 key frame header is complex (segmentation, loop filter, etc.
	// before the frame size fields), so our simple parser often returns
	// garbage for non-trivial headers. Sanity-check the result.
	width := w.width
	height := w.height
	if width < 16 || height < 16 || width > 7680 || height > 4320 {
		width = 1280
		height = 720
		zap.S().Debugf("IVF header using default resolution %dx%d (parsed=%dx%d)",
			width, height, w.width, w.height)
	}

	hdr := make([]byte, 32)
	binary.LittleEndian.PutUint32(hdr[0:4], 0x46494B44)       // "DKIF"
	binary.LittleEndian.PutUint16(hdr[4:6], 0)                 // version
	binary.LittleEndian.PutUint16(hdr[6:8], 32)                // header length
	binary.LittleEndian.PutUint32(hdr[8:12], 0x30385056)       // "VP80"
	binary.LittleEndian.PutUint16(hdr[12:14], width)           // width
	binary.LittleEndian.PutUint16(hdr[14:16], height)          // height
	binary.LittleEndian.PutUint32(hdr[16:20], 1000)            // timebase denominator
	binary.LittleEndian.PutUint32(hdr[20:24], 1)               // timebase numerator  (1/1000 = ms)
	binary.LittleEndian.PutUint32(hdr[24:28], 0)               // frame count (filled at stop)
	binary.LittleEndian.PutUint32(hdr[28:32], 0)               // unused
	if _, err := w.file.Write(hdr); err != nil {
		zap.S().Warnf("IVF header write failed: %s", err)
	}
}

func (w *IVFRecorderWriter) writeIVFFrame(data []byte, timestamp uint64) {
	fhdr := make([]byte, 12)
	binary.LittleEndian.PutUint32(fhdr[0:4], uint32(len(data)))
	binary.LittleEndian.PutUint64(fhdr[4:12], timestamp)
	if _, err := w.file.Write(fhdr); err != nil {
		zap.S().Warnf("IVF frame header write failed: %s", err)
		return
	}
	if _, err := w.file.Write(data); err != nil {
		zap.S().Warnf("IVF frame data write failed: %s", err)
	}
}

func (w *IVFRecorderWriter) updateFrameCount() {
	if w.file == nil {
		return
	}
	pos, err := w.file.Seek(0, 1) // save current position
	if err != nil {
		return
	}
	// Rewind to frame count field (offset 24) and update it
	w.file.Seek(24, 0)
	fc := make([]byte, 4)
	binary.LittleEndian.PutUint32(fc, uint32(w.frameCount))
	w.file.Write(fc)
	w.file.Seek(pos, 0) // restore position
}

func (w *IVFRecorderWriter) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true

	// Flush any remaining VP8 frame
	if len(w.vp8Buf) > 0 {
		w.flushFrame()
	}

	// Update frame count in header
	w.updateFrameCount()

	file := w.file
	w.file = nil
	w.mu.Unlock()

	if file != nil {
		if err := file.Close(); err != nil {
			zap.S().Warnf("Close IVF file failed path=%s: %s", w.filePath, err)
			return err
		}
	}
	zap.S().Debugf("IVF file closed: %s frames=%d", w.filePath, w.frameCount)
	return nil
}

// parseVP8Resolution extracts video width/height from a VP8 key frame's bitstream.
// VP8 stores resolution in macroblock units (16px), the returned values are in pixels.
func parseVP8Resolution(data []byte) (width, height uint16, ok bool) {
	if len(data) < 10 || data[0]&0x01 != 0 {
		return 0, 0, false
	}
	// Try tag length 1-3 to find start code 0x9D, 0x01, 0x2A
	tagLen := -1
	for tl := 1; tl <= 3; tl++ {
		if tl+3 < len(data) && data[tl] == 0x9D && data[tl+1] == 0x01 && data[tl+2] == 0x2A {
			tagLen = tl
			break
		}
	}
	if tagLen < 0 {
		return 0, 0, false
	}
	// Bool decoder data starts after start code.
	// VP8 key frame header: color_space(1) + clamp_type(1) + h_size(14) + v_size(14)
	bd := newVP8BitReader(data[tagLen+3:])
	_ = bd.readBit128()      // color_space / horizontal_scale
	mbs := bd.readLiteral14() // horizontal_size in macroblocks
	_ = bd.readBit128()       // clamp_type / vertical_scale
	mbs2 := bd.readLiteral14() // vertical_size in macroblocks

	if mbs <= 0 || mbs2 <= 0 {
		return 0, 0, false
	}

	// Convert macroblock units to pixels (each macroblock = 16x16)
	width = uint16(mbs) * 16
	height = uint16(mbs2) * 16
	return width, height, true
}

// vp8BitReader is a minimal VP8 boolean decoder for probability-128 reads
// (used for reading frame width/height from a VP8 key frame header).
type vp8BitReader struct {
	data []byte
	pos  int
	val  int
	rng  int
}

func newVP8BitReader(data []byte) *vp8BitReader {
	r := &vp8BitReader{data: data, rng: 255, val: 0}
	if len(data) >= 2 {
		r.val = (int(data[0]) << 8) | int(data[1])
		r.pos = 2
	}
	return r
}

// readBit128 reads one bit with probability 128 (simplified VP8 bool decoder).
func (r *vp8BitReader) readBit128() int {
	split := 1 + (((r.rng - 1) * 128) >> 8) // = 128 when rng=255
	var bit int
	if r.val < split {
		r.rng = split
		bit = 0
	} else {
		r.rng = r.rng - split
		r.val = r.val - split
		bit = 1
	}
	// Renormalize
	for r.rng < 128 {
		r.rng <<= 1
		if r.pos < len(r.data) {
			r.val = (r.val << 1) | int(r.data[r.pos])
			r.pos++
		} else {
			r.val <<= 1
		}
	}
	return bit
}

// readLiteral14 reads a 14-bit literal with probability 128 per bit, MSB first.
func (r *vp8BitReader) readLiteral14() int {
	result := 0
	for i := 0; i < 14; i++ {
		result = (result << 1) | r.readBit128()
	}
	return result
}

// Ensure Comply with RecorderWriter interface
var _ RecorderWriter = (*IVFRecorderWriter)(nil)

// ---------------------------------------------------------------------------
// TempOpusWriter — records Opus packets with timestamp info for later remux
// ---------------------------------------------------------------------------

// TempOpusWriter writes Opus packets to a temp binary file for later Ogg Opus muxing.
// Format: [8-byte timestamp_us][4-byte length][N-byte Opus data] repeated.
type TempOpusWriter struct {
	file     *os.File
	filePath string
	mu       sync.Mutex
	closed   bool

	audioBaseTS    int64
	lastFlushedTS  int64
}

func NewTempOpusWriter(path string) (*TempOpusWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create opus temp file: %w", err)
	}
	return &TempOpusWriter{file: f, filePath: path}, nil
}

func (w *TempOpusWriter) WriteRTP(pkt *rtp.Packet) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	// Timestamp relativization (Opus is 48kHz)
	ts := int64(pkt.Header.Timestamp)
	if w.audioBaseTS == 0 {
		w.audioBaseTS = ts
	}
	tsMs := (ts - w.audioBaseTS) * 1000 / 48000
	if tsMs <= w.lastFlushedTS {
		tsMs = w.lastFlushedTS + 1
	}
	w.lastFlushedTS = tsMs

	tsUs := tsMs * 1000 // convert to microseconds for Ogg Opus granule

	// Write: timestamp (us) + length + payload
	hdr := make([]byte, 12)
	binary.LittleEndian.PutUint64(hdr[0:8], uint64(tsUs))
	binary.LittleEndian.PutUint32(hdr[8:12], uint32(len(pkt.Payload)))

	if _, err := w.file.Write(hdr); err != nil {
		return err
	}
	if _, err := w.file.Write(pkt.Payload); err != nil {
		return err
	}
	return nil
}

func (w *TempOpusWriter) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	file := w.file
	w.file = nil
	w.mu.Unlock()

	if file != nil {
		return file.Close()
	}
	return nil
}

var _ RecorderWriter = (*TempOpusWriter)(nil)

// ---------------------------------------------------------------------------
// OggOpusMuxer — reads TempOpusWriter output and writes a valid Ogg Opus file
// ---------------------------------------------------------------------------

type opusPacket struct {
	tsUs int64 // timestamp in microseconds (48kHz sample clock)
	data []byte
}

func readOpusPackets(path string) ([]opusPacket, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var packets []opusPacket
	for {
		hdr := make([]byte, 12)
		_, err := f.Read(hdr)
		if err != nil {
			break
		}
		tsUs := int64(binary.LittleEndian.Uint64(hdr[0:8]))
		length := binary.LittleEndian.Uint32(hdr[8:12])
		if length == 0 {
			continue
		}
		data := make([]byte, length)
		if _, err := f.Read(data); err != nil {
			break
		}
		packets = append(packets, opusPacket{tsUs: tsUs, data: data})
	}
	return packets, nil
}

// WriteOggOpus creates an Ogg Opus file from the given packets.
// Output is a playable .opus file that ffmpeg can read for remuxing.
func WriteOggOpus(packets []opusPacket, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create ogg file: %w", err)
	}
	defer f.Close()

	serial := uint32(1)
	pageSeq := uint32(0)

	// ---- Page 0: BOS with ONLY OpusHead (per RFC 7845 §3) ----
	// OpusHead packet (19 bytes)
	opusHead := make([]byte, 19)
	copy(opusHead[0:8], []byte("OpusHead"))
	opusHead[8] = 1                     // version
	opusHead[9] = 2                     // channels (stereo)
	binary.LittleEndian.PutUint16(opusHead[10:12], 0) // pre-skip (0 for simplicity)
	binary.LittleEndian.PutUint32(opusHead[12:16], 48000) // input sample rate
	binary.LittleEndian.PutUint16(opusHead[16:18], 0) // output gain
	opusHead[18] = 0                    // mapping family

	// Write BOS page (OpusHead alone, per spec)
	writeOggPage(f, serial, &pageSeq, 0x02, 0, [][]byte{opusHead})

	// ---- Page 1: OpusTags (per RFC 7845 §3: second page) ----
	opusTags := make([]byte, 8+4)
	copy(opusTags[0:8], []byte("OpusTags"))
	binary.LittleEndian.PutUint32(opusTags[8:12], 0) // vendor string length (empty)

	writeOggPage(f, serial, &pageSeq, 0x00, 0, [][]byte{opusTags})

	if len(packets) == 0 {
		// Write EOS page for empty file
		writeOggPage(f, serial, &pageSeq, 0x04, 0, nil)
		return nil
	}

	// ---- Data pages ----
	// Accumulate packets per page (max 255 segments per Ogg page, < 64KB data)
	const maxPageData = 64 * 1024
	const maxSegmentsPerPage = 255

	var batchPackets [][]byte
	var totalSize int
	var batchGranule int64

	for i, p := range packets {
		granule := p.tsUs * 48 / 1000
		pktData := p.data

		// Flush current batch if adding this packet would exceed limits
		if len(batchPackets) >= maxSegmentsPerPage ||
			(len(pktData)+totalSize > maxPageData && len(batchPackets) > 0) {
			writeOggPage(f, serial, &pageSeq, 0x00, batchGranule, batchPackets)
			batchPackets = nil
			totalSize = 0
		}

		batchPackets = append(batchPackets, pktData)
		totalSize += len(pktData)
		batchGranule = granule

		// Write page when limits reached or this is the last packet
		isLast := i == len(packets)-1
		if isLast || totalSize > maxPageData || len(batchPackets) >= maxSegmentsPerPage {
			flags := 0x00
			if isLast {
				flags = 0x04 // EOS
			}
			writeOggPage(f, serial, &pageSeq, flags, batchGranule, batchPackets)
			batchPackets = nil
			totalSize = 0
		}
	}

	return nil
}

func writeOggPage(f *os.File, serial uint32, seq *uint32, flags int, granule int64, packets [][]byte) {
	// Build segment table
	segTable := make([]byte, 0, len(packets))
	packetData := make([]byte, 0)
	for _, pkt := range packets {
		packetData = append(packetData, pkt...)
		// Each packet is one segment; if > 255 bytes, split into 255-byte segments
		remaining := len(pkt)
		for remaining > 0 {
			if remaining >= 255 {
				segTable = append(segTable, 255)
				remaining -= 255
			} else {
				segTable = append(segTable, byte(remaining))
				remaining = 0
			}
		}
	}

	// Page header (27 bytes)
	page := make([]byte, 0, 27+len(segTable)+len(packetData))
	page = append(page, 0x4F, 0x67, 0x67, 0x53) // "OggS"
	page = append(page, 0)                        // stream version
	page = append(page, byte(flags))              // header type flag
	granBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(granBuf, uint64(granule))
	page = append(page, granBuf...)               // granule position
	serialBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(serialBuf, serial)
	page = append(page, serialBuf...)              // bitstream serial
	seqBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(seqBuf, *seq)
	page = append(page, seqBuf...)                  // page sequence number
	*seq++

	// CRC placeholder (4 bytes, set to 0 for calculation)
	crcBuf := make([]byte, 4)
	page = append(page, crcBuf...)                  // CRC (0 for now)

	page = append(page, byte(len(segTable)))        // number of page segments
	page = append(page, segTable...)                // segment table
	page = append(page, packetData...)              // packet data

	// Calculate CRC over the whole page (with CRC field = 0)
	crc := oggCRC(page)
	page[22] = byte(crc)
	page[23] = byte(crc >> 8)
	page[24] = byte(crc >> 16)
	page[25] = byte(crc >> 24)

	f.Write(page)
}

// oggCRC32 table for Ogg CRC-32 (non-reflected, polynomial 0x04C11DB7)
// Ogg spec requires MSB-first CRC, not the common reflected CRC-32.
var oggCRCTab [256]uint32

func init() {
	for i := 0; i < 256; i++ {
		c := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if c&0x80000000 != 0 {
				c = (c << 1) ^ 0x04C11DB7
			} else {
				c <<= 1
			}
		}
		oggCRCTab[i] = c
	}
}

func oggCRC(data []byte) uint32 {
	c := uint32(0)
	for _, b := range data {
		c = (c << 8) ^ oggCRCTab[byte(c>>24)^b]
	}
	return c
}

// ---------------------------------------------------------------------------
// ClientRecorder — holds the video (IVF) + audio (temp) for one participant
// ---------------------------------------------------------------------------

// ClientRecorder manages the temporary recording files for a single participant.
// At Stop(), it muxes IVF + OggOpus into a final WebM via ffmpeg.
type ClientRecorder struct {
	clientID string

	videoWriter *IVFRecorderWriter
	audioWriter *TempOpusWriter

	videoPath string // temp .ivf file
	audioPath string // temp .opus.raw file
	outputPath string // final .webm file (what DB points to)

	hasVideo bool
	hasAudio bool
}

// Remux produces the final .webm from temp files using ffmpeg.
func (cr *ClientRecorder) Remux(ffmpegPath string) error {
	start := time.Now()
	defer func() {
		zap.S().Infof("Remux completed in %v: %s", time.Since(start), cr.outputPath)
	}()

	// Step 1: Close writers first to flush all data
	if cr.videoWriter != nil {
		cr.videoWriter.Close()
	}
	if cr.audioWriter != nil {
		cr.audioWriter.Close()
	}

	// Step 2: Create Ogg Opus from raw audio
	oggPath := cr.outputPath + ".tmp.opus"
	if cr.hasAudio && fileExists(cr.audioPath) {
		packets, err := readOpusPackets(cr.audioPath)
		if err == nil && len(packets) > 0 {
			if err := WriteOggOpus(packets, oggPath); err != nil {
				zap.S().Warnf("WriteOggOpus failed: %s", err)
				_ = os.Remove(oggPath)
				oggPath = ""
			}
		} else {
			oggPath = ""
		}
	} else {
		oggPath = ""
	}

	// Step 3: Mux IVF + Ogg Opus → WebM via ffmpeg
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	args := []string{"-hide_banner", "-loglevel", "error", "-y"}
	if cr.hasVideo && fileExists(cr.videoPath) {
		args = append(args, "-i", cr.videoPath)
	}
	if oggPath != "" && fileExists(oggPath) {
		args = append(args, "-i", oggPath)
	}

	if len(args) <= 4 {
		// No inputs at all — nothing to do
		return nil
	}

	args = append(args, "-map", "0")
	if oggPath != "" {
		args = append(args, "-map", "1")
	}
	args = append(args, "-c", "copy")
	// If we have both inputs, flag the output as WebM
	if oggPath != "" {
		args = append(args, "-f", "webm")
	}
	args = append(args, cr.outputPath)

	zap.S().Infof("Remux cmd: %s %v", ffmpegPath, args)
	cmd := exec.Command(ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(cr.outputPath)
		return fmt.Errorf("ffmpeg remux failed: %s output=%s", err, strings.TrimSpace(string(output)))
	}

	// Step 4: Clean up temp files
	_ = os.Remove(cr.videoPath)
	_ = os.Remove(cr.audioPath)
	if oggPath != "" {
		_ = os.Remove(oggPath)
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
