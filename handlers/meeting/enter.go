package meeting

type Meeting struct{}

type CreateMeetingRequest struct {
	Title    string `json:"title" binding:"required" display:"会议标题"`
	Password string `json:"password" display:"会议密码"`
}

type JoinMeetingRequest struct {
	Password string `json:"password" display:"会议密码"`
}

type MeetingVO struct {
	ID        uint   `json:"id"`
	RoomNo    uint   `json:"roomNo"`
	Title     string `json:"title"`
	HostID    uint   `json:"hostId"`
	HostName  string `json:"hostName"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}
