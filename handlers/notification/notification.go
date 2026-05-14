package notification

import (
	"fast-gin/dal/query"
	"fast-gin/global"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/service/common"
	"fast-gin/utils/res"
	"time"

	"github.com/gin-gonic/gin"
)

// NotifPusher is an interface for pushing real-time notifications to users.
// Injected from routers to avoid circular imports.
type NotifPusher interface {
	Push(userID uint, v any)
}

type CreateNotificationRequest struct {
	ToUserID uint   `json:"toUserId" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Message  string `json:"message"`
}

var notifPusher NotifPusher

func SetNotifPusher(p NotifPusher) {
	notifPusher = p
}

func (Notification) CreateView(c *gin.Context) {
	req := middleware.GetJSON[CreateNotificationRequest](c)
	fromUserID := middleware.GetUserID(c)
	if fromUserID == 0 {
		res.FailAuth(c)
		return
	}
	if fromUserID == req.ToUserID {
		res.FailWithMsg(c, "不能给自己发送通知")
		return
	}

	notification := models.Notification{
		FromUserID: fromUserID,
		ToUserID:   req.ToUserID,
		Type:       req.Type,
		Message:    req.Message,
		Status:     "unread",
	}
	if err := query.Notification.WithContext(c).Create(&notification); err != nil {
		res.FailWithMsg(c, "创建通知失败")
		return
	}

	// Push real-time notification via WebSocket if the user is online
	if notifPusher != nil {
		notifPusher.Push(req.ToUserID, map[string]any{
			"type":         "new-notification",
			"notification": notification,
		})
		count, err := query.Notification.WithContext(c).
			Where(query.Notification.ToUserID.Eq(req.ToUserID), query.Notification.Status.Eq("unread")).
			Count()
		if err == nil {
			notifPusher.Push(req.ToUserID, map[string]any{
				"type":  "unread-count",
				"count": count,
			})
		}
	}

	res.OkWithData(c, notification)
}

func (Notification) ListView(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		res.FailAuth(c)
		return
	}

	pageInfo := middleware.GetQuery[models.PageInfo](c)
	if pageInfo.Limit < 1 || pageInfo.Limit > 100 {
		pageInfo.Limit = 20
	}

	list, count, err := common.QueryList(models.Notification{}, common.QueryOption{
		PageInfo: pageInfo,
		Where:    global.DB.Where(query.Notification.ToUserID.Eq(userID)),
	})
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithList(c, list, count)
}

func (Notification) UnreadCountView(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		res.FailAuth(c)
		return
	}
	count, err := query.Notification.WithContext(c).
		Where(query.Notification.ToUserID.Eq(userID), query.Notification.Status.Eq("unread")).
		Count()
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	res.OkWithData(c, count)
}

func (Notification) ReadView(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		res.FailAuth(c)
		return
	}
	uri := middleware.GetUri[models.BindId](c)
	now := time.Now()
	result, err := query.Notification.WithContext(c).
		Where(query.Notification.ID.Eq(uri.ID), query.Notification.ToUserID.Eq(userID)).
		UpdateSimple(query.Notification.Status.Value("read"), query.Notification.ReadAt.Value(now))
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	if result.RowsAffected == 0 {
		res.FailWithMsg(c, "通知不存在")
		return
	}

	// Push updated unread count
	if notifPusher != nil {
		count, err := query.Notification.WithContext(c).
			Where(query.Notification.ToUserID.Eq(userID), query.Notification.Status.Eq("unread")).
			Count()
		if err == nil {
			notifPusher.Push(userID, map[string]any{
				"type":  "unread-count",
				"count": count,
			})
		}
	}

	res.OkWithMsg(c, "已读")
}

func (Notification) ReadAllView(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		res.FailAuth(c)
		return
	}
	now := time.Now()
	_, err := query.Notification.WithContext(c).
		Where(query.Notification.ToUserID.Eq(userID), query.Notification.Status.Eq("unread")).
		UpdateSimple(query.Notification.Status.Value("read"), query.Notification.ReadAt.Value(now))
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	// Push updated unread count
	if notifPusher != nil {
		notifPusher.Push(userID, map[string]any{
			"type":  "unread-count",
			"count": 0,
		})
	}

	res.OkWithMsg(c, "全部已读")
}

func (Notification) DeleteView(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		res.FailAuth(c)
		return
	}
	uri := middleware.GetUri[models.BindId](c)
	result, err := query.Notification.WithContext(c).
		Where(query.Notification.ID.Eq(uri.ID), query.Notification.ToUserID.Eq(userID)).
		Delete()
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	if result.RowsAffected == 0 {
		res.FailWithMsg(c, "通知不存在")
		return
	}

	// Push updated unread count
	if notifPusher != nil {
		count, err := query.Notification.WithContext(c).
			Where(query.Notification.ToUserID.Eq(userID), query.Notification.Status.Eq("unread")).
			Count()
		if err == nil {
			notifPusher.Push(userID, map[string]any{
				"type":  "unread-count",
				"count": count,
			})
		}
	}

	res.OkWithMsg(c, "已删除")
}
