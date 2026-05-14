package meeting

import (
	"errors"
	"fast-gin/dal/query"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/utils/pwd"
	"fast-gin/utils/res"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (Meeting) JoinView(c *gin.Context) {
	req := middleware.GetJSON[JoinMeetingRequest](c)
	claims := middleware.GetAuth(c)
	roomNo := middleware.GetUri[models.BindRoomNo](c).RoomNo

	meeting, err := query.Meeting.WithContext(c).Where(query.Meeting.RoomNo.Eq(roomNo)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			res.FailWithMsg(c, "会议不存在")
			return
		}
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	if meeting.Status == "ended" {
		res.FailWithMsg(c, "会议已结束")
		return
	}

	if meeting.Password != "" {
		if req.Password == "" {
			res.FailWithMsg(c, "需要会议密码")
			return
		}
		if ok := pwd.CompareHashAndPassword(meeting.Password, req.Password); !ok {
			res.FailWithMsg(c, "会议密码错误")
			return
		}
	}

	// Record participant join
	// Fetch user display name
	displayName := ""
	user, err := query.User.WithContext(c).Where(query.User.ID.Eq(claims.UserID)).First()
	if err == nil {
		if user.Nickname != "" {
			displayName = user.Nickname
		} else if user.RealName != "" {
			displayName = user.RealName
		} else {
			displayName = user.Username
		}
	}

	participant := &models.MeetingParticipant{
		MeetingID:   meeting.ID,
		UserID:      claims.UserID,
		DisplayName: displayName,
		JoinedAt:    time.Now(),
		IsHost:      meeting.HostID == claims.UserID,
	}
	if err := query.MeetingParticipant.WithContext(c).Create(participant); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithData(c, meeting.RoomNo)
}
