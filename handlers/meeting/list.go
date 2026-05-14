package meeting

import (
	"fast-gin/dal/query"
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

	// Get meetings the user participated in
	participatedIDs := make([]uint, 0)
	query.MeetingParticipant.WithContext(c).
		Where(query.MeetingParticipant.UserID.Eq(claims.UserID)).
		Pluck(query.MeetingParticipant.MeetingID, &participatedIDs)

	// Build OR condition: host_id = userID OR id IN (participated meetings)
	where := global.DB.Where(query.Meeting.HostID.Eq(claims.UserID))
	if len(participatedIDs) > 0 {
		where = where.Or(query.Meeting.ID.In(participatedIDs...))
	}

	list, count, err := common.QueryList(models.Meeting{}, common.QueryOption{
		PageInfo: page,
		Where:    where,
	})
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithList(c, list, count)
}
