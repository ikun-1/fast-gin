package models

import "time"

type MeetingParticipant struct {
	Model
	MeetingID   uint       `gorm:"not null;index;comment:会议ID" json:"meetingId"`
	UserID      uint       `gorm:"not null;comment:用户ID" json:"userId"`
	DisplayName string     `gorm:"size:64;comment:参会显示名" json:"displayName"`
	JoinedAt    time.Time  `gorm:"comment:加入时间" json:"joinedAt"`
	LeftAt      *time.Time `gorm:"comment:离开时间" json:"leftAt"`
	IsHost      bool       `gorm:"default:false;comment:是否主持人" json:"isHost"`
}

func init() {
	MigrateModels = append(MigrateModels, &MeetingParticipant{})
}
