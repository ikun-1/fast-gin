package models

import "time"

type MeetingQualitySnapshot struct {
	Model
	MeetingID      uint      `gorm:"index;not null;comment:会议ID" json:"meetingId"`
	UserID         uint      `gorm:"index;not null;comment:用户ID" json:"userId"`
	ClientID       string    `gorm:"size:128;index;comment:客户端ID" json:"clientId"`
	Label          string    `gorm:"size:32;index;comment:标签(audio/video/connection)" json:"label"`
	BytesSent      int64     `gorm:"comment:累计发送字节数" json:"bytesSent"`
	BytesReceived  int64     `gorm:"comment:累计接收字节数" json:"bytesReceived"`
	PacketsSent    int64     `gorm:"comment:累计发送包数" json:"packetsSent"`
	PacketsReceived int64    `gorm:"comment:累计接收包数" json:"packetsReceived"`
	PacketsLost    int64     `gorm:"comment:累计丢包数" json:"packetsLost"`
	JitterMs       float64   `gorm:"comment:抖动(ms)" json:"jitterMs"`
	RoundTripMs    float64   `gorm:"comment:往返时延(ms)" json:"roundTripMs"`
	BitrateKbps    float64   `gorm:"comment:估算比特率(kbps)" json:"bitrateKbps"`
	FrameWidth     int       `gorm:"comment:视频宽度" json:"frameWidth"`
	FrameHeight    int       `gorm:"comment:视频高度" json:"frameHeight"`
	FPS            float64   `gorm:"comment:帧率" json:"fps"`
	FramesDecoded  int       `gorm:"comment:已解码帧数" json:"framesDecoded"`
	TotalFramesLost int      `gorm:"comment:总丢失帧数" json:"totalFramesLost"`
	CandidateType  string    `gorm:"size:32;comment:ICE候选类型(host/srflx/relay)" json:"candidateType"`
	SnapshotAt     time.Time `gorm:"index;comment:采集时间" json:"snapshotAt"`
}

func init() {
	MigrateModels = append(MigrateModels, &MeetingQualitySnapshot{})
}
