package models

type MeetingInvitation struct {
	Model
	RoomID      uint   `json:"roomId" gorm:"not null;index;comment:会议ID"`
	SenderID    uint   `json:"senderId" gorm:"not null;index;comment:邀请发送者ID"`
	ReceiverID  uint   `json:"receiverId" gorm:"not null;index;comment:被邀请者ID"`
	InviteCode  string `json:"inviteCode" gorm:"size:64;not null;uniqueIndex;comment:邀请码"`
	Status      int8   `json:"status" gorm:"not null;default:0;comment:邀请状态 0待接受 1已接受 2已拒绝 3已过期"`
	ExpiredAt   int64  `json:"expiredAt" gorm:"comment:过期时间戳"`
}

func init() {
	MigrateModels = append(MigrateModels, &MeetingInvitation{})
}
