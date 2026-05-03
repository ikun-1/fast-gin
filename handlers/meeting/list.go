package meeting

import (
	"fast-gin/global"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/service/common"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
)

func (Meeting) ListView(c *gin.Context) {
	claims := middleware.GetAuth(c)
	page := middleware.GetQuery[models.PageInfo](c)

	meetings, count, err := common.QueryList(models.Meeting{}, common.QueryOption{
		PageInfo: page,
		Where:    global.DB.Where("host_id = ?", claims.UserID).
			Or("id IN (?)", global.DB.Model(&models.MeetingParticipant{}).
				Select("meeting_id").
				Where("user_id = ?", claims.UserID)),
	})
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithList(c, meetings, count)
}
