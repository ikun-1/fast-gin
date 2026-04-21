package models

type SignalMessage struct {
	Model
	RoomID         uint   `json:"roomId" gorm:"not null;index;comment:会议ID"`
	SenderID       uint   `json:"senderId" gorm:"not null;index;comment:发送者ID"`
	MessageType    string `json:"messageType" gorm:"size:50;not null;comment:消息类型"`
	MessageContent string `json:"messageContent" gorm:"type:text;comment:消息内容"`
}

func init() {
	MigrateModels = append(MigrateModels, &SignalMessage{})
}
