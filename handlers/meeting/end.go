package meeting

import (
	"errors"
	"fast-gin/dal/query"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/service/ws_serv"
	"fast-gin/utils/res"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (Meeting) EndView(c *gin.Context) {
	uri := middleware.GetUri[models.BindRoomNo](c)
	claims := middleware.GetAuth(c)

	meeting, err := query.Meeting.WithContext(c).Where(query.Meeting.RoomNo.Eq(uri.RoomNo)).First()
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

	// Stop active recording if any (async — remux via ffmpeg can take seconds)
	if ws_serv.GlobalRecordingManager.IsRecording(uri.RoomNo) {
		go ws_serv.GlobalRecordingManager.StopSession(uri.RoomNo)
	}

	now := time.Now()
	if _, err := query.Meeting.WithContext(c).
		Where(query.Meeting.ID.Eq(meeting.ID)).
		UpdateSimple(query.Meeting.Status.Value("ended"), query.Meeting.EndedAt.Value(now)); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	// Update all participants' left_at if not already set
	query.MeetingParticipant.WithContext(c).
		Where(query.MeetingParticipant.MeetingID.Eq(meeting.ID), query.MeetingParticipant.LeftAt.IsNull()).
		Update(query.MeetingParticipant.LeftAt, &now)

	res.OkSuccess(c)
}
