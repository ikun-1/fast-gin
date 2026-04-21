package models

type MeetingActionLog struct {
	Model
	RoomID   uint   `json:"roomId" gorm:"not null;index;comment:会议ID"`
	UserID   uint   `json:"userId" gorm:"not null;index;comment:操作用户ID"`
	Action   string `json:"action" gorm:"size:100;not null;comment:动作名称"`
	Detail   string `json:"detail" gorm:"type:text;comment:动作详情"`
}

func init() {
	MigrateModels = append(MigrateModels, &MeetingActionLog{})
}
