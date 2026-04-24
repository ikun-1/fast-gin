package meeting

import (
	"errors"
	"fast-gin/dal/query"
	"fast-gin/global"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (Meeting) InfoView(c *gin.Context) {
	uri := middleware.GetUri[models.BindRoomNo](c)

	var meeting models.Meeting
	err := global.DB.WithContext(c).Where("room_no = ?", uri.RoomNo).First(&meeting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			res.FailWithMsg(c, "会议不存在")
			return
		}
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	host, err := query.User.WithContext(c).
		Where(query.User.ID.Eq(meeting.HostID)).
		Select(query.User.Nickname, query.User.Username).
		First()
	hostName := ""
	if err == nil {
		if host.Nickname != "" {
			hostName = host.Nickname
		} else {
			hostName = host.Username
		}
	}

	vo := MeetingVO{
		ID:        meeting.ID,
		RoomNo:    meeting.RoomNo,
		Title:     meeting.Title,
		HostID:    meeting.HostID,
		HostName:  hostName,
		Status:    meeting.Status,
		CreatedAt: meeting.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	res.OkWithData(c, vo)
}
