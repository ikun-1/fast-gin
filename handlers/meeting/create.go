package meeting

import (
	"fast-gin/global"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/utils/pwd"
	"fast-gin/utils/res"
	"math/rand"

	"github.com/gin-gonic/gin"
)

func (Meeting) CreateView(c *gin.Context) {
	req := middleware.GetJSON[CreateMeetingRequest](c)
	claims := middleware.GetAuth(c)

	// Generate a unique 6-digit room number
	var roomNo uint
	for {
		roomNo = uint(rand.Intn(900000) + 100000)
		var count int64
		global.DB.WithContext(c).Model(&models.Meeting{}).Where("room_no = ?", roomNo).Count(&count)
		if count == 0 {
			break
		}
	}

	meeting := &models.Meeting{
		RoomNo: roomNo,
		Title:  req.Title,
		HostID: claims.UserID,
		Status: "waiting",
	}
	if req.Password != "" {
		hash := pwd.GenerateFromPassword(req.Password)
		if hash == "" {
			res.FailWithMsg(c, "密码加密失败")
			return
		}
		meeting.Password = hash
	}

	if err := global.DB.WithContext(c).Create(meeting).Error; err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithData(c, meeting.RoomNo)
}
