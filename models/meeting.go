package models

import "time"

type Meeting struct {
	Model
	RoomNo    uint       `gorm:"uniqueIndex;not null;comment:会议房间号" json:"roomNo"`
	Title     string     `gorm:"size:128;comment:会议标题" json:"title"`
	Password  string     `gorm:"size:128;comment:会议密码(哈希)" json:"-"`
	HostID    uint       `gorm:"not null;index;comment:主持人用户ID" json:"hostId"`
	Status    string     `gorm:"size:20;default:waiting;comment:waiting/active/ended" json:"status"`
	StartedAt *time.Time `gorm:"comment:会议开始时间" json:"startedAt"`
	EndedAt   *time.Time `gorm:"comment:会议结束时间" json:"endedAt"`
}

func init() {
	MigrateModels = append(MigrateModels, &Meeting{})
}
