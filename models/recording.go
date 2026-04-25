package models

import "time"

type Recording struct {
	Model
	MeetingID   uint       `gorm:"index;not null;comment:关联会议ID" json:"meetingId"`
	RoomNo      uint       `gorm:"index;not null;comment:会议房间号" json:"roomNo"`
	HostID      uint       `gorm:"not null;comment:主持人用户ID" json:"hostId"`
	StartedAt   time.Time  `gorm:"not null;comment:录制开始时间" json:"startedAt"`
	EndedAt     *time.Time `gorm:"comment:录制结束时间" json:"endedAt"`
	DurationMs  int64      `gorm:"default:0;comment:录制时长(毫秒)" json:"durationMs"`
	Status      string     `gorm:"size:20;default:recording;comment:recording/completed/failed" json:"status"`
	FileCount   int        `gorm:"default:0;comment:文件数量" json:"fileCount"`
	StoragePath string     `gorm:"size:256;comment:录制存储目录" json:"storagePath"`
	UserID      uint       `gorm:"index;not null;comment:上传用户ID" json:"userId"`
	FileName    string     `gorm:"size:255;comment:原始文件名" json:"fileName"`
	FilePath    string     `gorm:"size:512;comment:文件存储路径" json:"filePath"`
	FileSize    int64      `gorm:"comment:文件大小(字节)" json:"fileSize"`
	Duration    float64    `gorm:"comment:录制时长(秒)" json:"duration"`
}

type RecordingFile struct {
	Model
	RecordingID uint   `gorm:"index;not null;comment:录制ID" json:"recordingId"`
	ClientID    string `gorm:"size:128;index;not null;comment:客户端ID" json:"clientId"`
	UserID      uint   `gorm:"index;not null;comment:用户ID" json:"userId"`
	DisplayName string `gorm:"size:128;comment:显示名称" json:"displayName"`
	FilePath    string `gorm:"size:512;not null;comment:文件路径" json:"filePath"`
	Kind        string `gorm:"size:32;comment:文件类型" json:"kind"`
	Codec       string `gorm:"size:64;comment:编码格式" json:"codec"`
	FileSize    int64  `gorm:"default:0;comment:文件大小" json:"fileSize"`
}

func init() {
	MigrateModels = append(MigrateModels, &Recording{}, &RecordingFile{})
}
