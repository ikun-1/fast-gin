package models

type MeetingRoomMember struct {
	Model
	RoomID       uint  `json:"roomId" gorm:"not null;index;comment:会议ID"`
	UserID       uint  `json:"userId" gorm:"not null;index;comment:用户ID"`
	MemberStatus int8  `json:"memberStatus" gorm:"not null;default:0;comment:成员状态 0在会 1已离开"`
	JoinTime     int64 `json:"joinTime" gorm:"comment:加入时间戳"`
	LeaveTime    int64 `json:"leaveTime" gorm:"comment:离开时间戳"`
}

func init() {
	MigrateModels = append(MigrateModels, &MeetingRoomMember{})
}
