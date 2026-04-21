package room_serv

type RoomService struct{}

type MeetingRoomVO struct {
	ID         uint   `json:"id"`
	RoomNo     string `json:"roomNo"`
	Title      string `json:"title"`
	CreatorID  uint   `json:"creatorId"`
	RoomStatus int8   `json:"roomStatus"`
	MaxMembers int    `json:"maxMembers"`
}

type MeetingHostVO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
}

type MeetingMemberVO struct {
	ID           uint   `json:"id"`
	UserID       uint   `json:"userId"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname,omitempty"`
	Avatar       string `json:"avatar,omitempty"`
	MemberStatus int8   `json:"memberStatus"`
	JoinTime     int64  `json:"joinTime"`
	LeaveTime    int64  `json:"leaveTime"`
}

type MeetingMessageVO struct {
	ID             uint   `json:"id"`
	RoomID         uint   `json:"roomId"`
	SenderID       uint   `json:"senderId"`
	SenderName     string `json:"senderName,omitempty"`
	MessageType    string `json:"messageType"`
	MessageContent string `json:"messageContent"`
	CreatedAt      int64  `json:"createdAt"`
}

type MeetingAttachmentVO struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"`
	FileType string `json:"fileType"`
}

type MeetingDetailResponse struct {
	Room        MeetingRoomVO          `json:"room"`
	Host        MeetingHostVO          `json:"host"`
	Members     []MeetingMemberVO      `json:"members"`
	MyRole      string                 `json:"myRole"`
	Messages    []MeetingMessageVO     `json:"messages"`
	Attachments []MeetingAttachmentVO  `json:"attachments"`
}

var Service = new(RoomService)
