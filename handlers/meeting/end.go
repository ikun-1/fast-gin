package meeting

import (
	"errors"
	"fast-gin/global"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/utils/res"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (Meeting) EndView(c *gin.Context) {
	uri := middleware.GetUri[models.BindRoomNo](c)
	claims := middleware.GetAuth(c)

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

	if meeting.HostID != claims.UserID {
		res.FailPermission(c)
		return
	}

	now := time.Now()
	if err := global.DB.WithContext(c).Model(&meeting).
		Updates(map[string]any{"status": "ended", "ended_at": &now}).Error; err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkSuccess(c)
}
