package models

type MeetingAttachment struct {
	Model
	RoomID   uint   `json:"roomId" gorm:"not null;index;comment:会议ID"`
	UploaderID uint `json:"uploaderId" gorm:"not null;index;comment:上传者ID"`
	FileID   uint   `json:"fileId" gorm:"not null;index;comment:文件ID"`
	Title    string `json:"title" gorm:"size:255;comment:附件标题"`
	FileType string `json:"fileType" gorm:"size:50;comment:文件类型"`
}

func init() {
	MigrateModels = append(MigrateModels, &MeetingAttachment{})
}
