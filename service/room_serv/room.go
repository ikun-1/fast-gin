package room_serv

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"fast-gin/dal/query"
	"fast-gin/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (RoomService) CreateRoom(c *gin.Context, title string, maxMembers int, creatorID uint) (*models.MeetingRoom, error) {
	if maxMembers <= 0 {
		maxMembers = 2
	}

	room := &models.MeetingRoom{
		RoomNo:     generateRoomNo(),
		Title:      title,
		CreatorID:  creatorID,
		RoomStatus: 0,
		MaxMembers: maxMembers,
	}
	if err := query.MeetingRoom.WithContext(c).Create(room); err != nil {
		return nil, err
	}
	return room, nil
}

func (RoomService) JoinRoom(c *gin.Context, roomNo string, userID uint) (*models.MeetingRoom, error) {
	roomNo = strings.TrimSpace(roomNo)
	if roomNo == "" {
		return nil, errors.New("会议号不能为空")
	}

	room, err := query.MeetingRoom.WithContext(c).Where(query.MeetingRoom.RoomNo.Eq(roomNo)).Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("会议不存在")
		}
		return nil, err
	}

	memberQ := query.MeetingRoomMember.WithContext(c)
	if _, err := memberQ.Where(
		query.MeetingRoomMember.RoomID.Eq(room.ID),
		query.MeetingRoomMember.UserID.Eq(userID),
		query.MeetingRoomMember.MemberStatus.Eq(0),
	).Take(); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		member := &models.MeetingRoomMember{RoomID: room.ID, UserID: userID, MemberStatus: 0, JoinTime: time.Now().Unix()}
		if err := memberQ.Create(member); err != nil {
			return nil, err
		}
	}
	return room, nil
}

func (RoomService) GetMeetingDetail(c *gin.Context, roomID uint, myUserID uint) (*MeetingDetailResponse, error) {
	room, err := query.MeetingRoom.WithContext(c).Where(query.MeetingRoom.ID.Eq(roomID)).Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("会议不存在")
		}
		return nil, err
	}

	host, _ := query.User.WithContext(c).Where(query.User.ID.Eq(room.CreatorID)).Take()
	members, err := query.MeetingRoomMember.WithContext(c).Where(query.MeetingRoomMember.RoomID.Eq(room.ID)).Find()
	if err != nil {
		return nil, err
	}
	messages, err := query.SignalMessage.WithContext(c).Where(query.SignalMessage.RoomID.Eq(room.ID)).Order(query.SignalMessage.CreatedAt.Desc()).Find()
	if err != nil {
		return nil, err
	}
	attachments, err := query.MeetingAttachment.WithContext(c).Where(query.MeetingAttachment.RoomID.Eq(room.ID)).Find()
	if err != nil {
		return nil, err
	}

	roomVO := MeetingRoomVO{
		ID:         room.ID,
		RoomNo:     room.RoomNo,
		Title:      room.Title,
		CreatorID:  room.CreatorID,
		RoomStatus: room.RoomStatus,
		MaxMembers: room.MaxMembers,
	}

	hostVO := MeetingHostVO{}
	if host != nil {
		hostVO = MeetingHostVO{ID: host.ID, Username: host.Username, Nickname: host.Nickname}
	}

	memberVOs := make([]MeetingMemberVO, 0, len(members))
	for _, m := range members {
		user, _ := query.User.WithContext(c).Where(query.User.ID.Eq(m.UserID)).Take()
		memberVOs = append(memberVOs, MeetingMemberVO{
			ID:           m.ID,
			UserID:       m.UserID,
			Username:     func() string { if user != nil { return user.Username }; return "" }(),
			Nickname:     func() string { if user != nil { return user.Nickname }; return "" }(),
			MemberStatus: m.MemberStatus,
			JoinTime:     m.JoinTime,
			LeaveTime:    m.LeaveTime,
		})
	}

	messageVOs := make([]MeetingMessageVO, 0, len(messages))
	for _, msg := range messages {
		user, _ := query.User.WithContext(c).Where(query.User.ID.Eq(msg.SenderID)).Take()
		messageVOs = append(messageVOs, MeetingMessageVO{
			ID:             msg.ID,
			RoomID:         msg.RoomID,
			SenderID:       msg.SenderID,
			SenderName:     func() string { if user != nil { return user.Username }; return "" }(),
			MessageType:    msg.MessageType,
			MessageContent: msg.MessageContent,
			CreatedAt:      msg.CreatedAt.Unix(),
		})
	}

	attachmentVOs := make([]MeetingAttachmentVO, 0, len(attachments))
	for _, a := range attachments {
		attachmentVOs = append(attachmentVOs, MeetingAttachmentVO{ID: a.ID, Title: a.Title, FileType: a.FileType})
	}

	myRole := "member"
	if myUserID == room.CreatorID {
		myRole = "host"
	}

	return &MeetingDetailResponse{
		Room:        roomVO,
		Host:        hostVO,
		Members:     memberVOs,
		MyRole:      myRole,
		Messages:    messageVOs,
		Attachments: attachmentVOs,
	}, nil
}

func generateRoomNo() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", r.Intn(1000000))
}
