package models

type MeetingRoom struct {
	Model
	RoomNo     string `json:"roomNo" gorm:"size:32;not null;uniqueIndex;comment:会议号"`
	Title      string `json:"title" gorm:"size:100;not null;comment:会议标题"`
	CreatorID  uint   `json:"creatorId" gorm:"not null;index;comment:创建者ID"`
	RoomStatus int8   `json:"roomStatus" gorm:"not null;default:0;comment:会议状态 0未开始 1进行中 2已结束"`
	MaxMembers int    `json:"maxMembers" gorm:"not null;default:2;comment:最大成员数"`
}

func init() {
	MigrateModels = append(MigrateModels, &MeetingRoom{})
}
