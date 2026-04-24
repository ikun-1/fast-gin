package meeting

import (
	"errors"
	"fast-gin/global"
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

	var meeting models.Meeting
	err := global.DB.WithContext(c).Where("room_no = ?", roomNo).First(&meeting).Error
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
	participant := &models.MeetingParticipant{
		MeetingID: meeting.ID,
		UserID:    claims.UserID,
		JoinedAt:  time.Now(),
		IsHost:    meeting.HostID == claims.UserID,
	}
	if err := global.DB.WithContext(c).Create(participant).Error; err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithData(c, meeting.RoomNo)
}
