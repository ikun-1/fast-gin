package models

type Recording struct {
	Model
	RoomID   uint   `json:"roomId" gorm:"not null;index;comment:会议ID"`
	UserID   uint   `json:"userId" gorm:"not null;index;comment:录制发起者ID"`
	Title    string `json:"title" gorm:"size:100;not null;comment:录制标题"`
	FileID   *uint  `json:"fileId,omitempty" gorm:"index;comment:录制文件ID"`
	Status   int8   `json:"status" gorm:"not null;default:0;comment:录制状态 0进行中 1已完成 2已失败"`
	Duration int64  `json:"duration" gorm:"comment:录制时长秒"`
	Size     int64  `json:"size" gorm:"comment:文件大小字节"`
}

func init() {
	MigrateModels = append(MigrateModels, &Recording{})
}
