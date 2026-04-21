package models

type Whiteboard struct {
	Model
	RoomID        uint   `json:"roomId" gorm:"not null;index;comment:会议ID"`
	CreatorID     uint   `json:"creatorId" gorm:"not null;index;comment:创建者ID"`
	Name          string `json:"name" gorm:"size:100;not null;comment:白板名称"`
	SnapshotFileID *uint  `json:"snapshotFileId,omitempty" gorm:"index;comment:快照文件ID"`
	DataJSON      string `json:"dataJson" gorm:"type:longtext;comment:白板数据JSON"`
}

func init() {
	MigrateModels = append(MigrateModels, &Whiteboard{})
}
