package models

import "time"

type Notification struct {
	Model
	FromUserID uint       `gorm:"not null;index;comment:发送者ID" json:"fromUserId"`
	ToUserID   uint       `gorm:"not null;index;comment:接收者ID" json:"toUserId"`
	Type       string     `gorm:"size:32;not null;comment:类型(invitation)" json:"type"`
	Message    string     `gorm:"size:256;comment:消息内容" json:"message"`
	Status     string     `gorm:"size:16;default:unread;comment:状态(unread/read)" json:"status"`
	ReadAt     *time.Time `gorm:"comment:已读时间" json:"readAt,omitempty"`
}

func init() {
	MigrateModels = append(MigrateModels, &Notification{})
}
