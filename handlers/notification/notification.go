package notification

import (
	"fast-gin/global"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/utils/res"
	"time"

	"github.com/gin-gonic/gin"
)

type CreateNotificationRequest struct {
	ToUserID uint   `json:"toUserId" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Message  string `json:"message"`
}

// NotifPusher is an interface for pushing real-time notifications to users.
// Injected from routers to avoid circular imports.
type NotifPusher interface {
	Push(userID uint, v any)
}

var notifPusher NotifPusher

func SetNotifPusher(p NotifPusher) {
	notifPusher = p
}

func getCurrentUserID(c *gin.Context) uint {
	cl := middleware.GetAuth(c)
	if cl == nil {
		return 0
	}
	return cl.UserID
}

func (Notification) CreateView(c *gin.Context) {
	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithMsg(c, "参数错误")
		return
	}
	fromUserID := getCurrentUserID(c)
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
	if err := global.DB.Create(&notification).Error; err != nil {
		res.FailWithMsg(c, "创建通知失败")
		return
	}

	// Push real-time notification via WebSocket if the user is online
	if notifPusher != nil {
		notifPusher.Push(req.ToUserID, map[string]any{
			"type":         "new-notification",
			"notification": notification,
		})
		var count int64
		global.DB.Model(&models.Notification{}).
			Where("to_user_id = ? AND status = ?", req.ToUserID, "unread").
			Count(&count)
		notifPusher.Push(req.ToUserID, map[string]any{
			"type":  "unread-count",
			"count": count,
		})
	}

	res.OkWithData(c, notification)
}

func (Notification) ListView(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID == 0 {
		res.FailAuth(c)
		return
	}

	page := 1
	limit := 20
	c.ShouldBindQuery(&struct {
		Page  int `form:"page"`
		Limit int `form:"limit"`
	}{Page: page, Limit: limit})
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var list []models.Notification
	var count int64
	global.DB.Model(&models.Notification{}).
		Where("to_user_id = ?", userID).
		Count(&count)

	global.DB.Where("to_user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&list)

	res.OkWithList(c, list, count)
}

func (Notification) UnreadCountView(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID == 0 {
		res.FailAuth(c)
		return
	}
	var count int64
	global.DB.Model(&models.Notification{}).
		Where("to_user_id = ? AND status = ?", userID, "unread").
		Count(&count)
	res.OkWithData(c, count)
}

func (Notification) ReadView(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID == 0 {
		res.FailAuth(c)
		return
	}
	var uri models.BindId
	if err := c.ShouldBindUri(&uri); err != nil {
		res.FailWithMsg(c, "参数错误")
		return
	}
	now := time.Now()
	result := global.DB.Model(&models.Notification{}).
		Where("id = ? AND to_user_id = ?", uri.ID, userID).
		Updates(map[string]any{
			"status":  "read",
			"read_at": &now,
		})
	if result.RowsAffected == 0 {
		res.FailWithMsg(c, "通知不存在")
		return
	}

	// Push updated unread count
	if notifPusher != nil {
		var count int64
		global.DB.Model(&models.Notification{}).
			Where("to_user_id = ? AND status = ?", userID, "unread").
			Count(&count)
		notifPusher.Push(userID, map[string]any{
			"type":  "unread-count",
			"count": count,
		})
	}

	res.OkWithMsg(c, "已读")
}

func (Notification) ReadAllView(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID == 0 {
		res.FailAuth(c)
		return
	}
	now := time.Now()
	global.DB.Model(&models.Notification{}).
		Where("to_user_id = ? AND status = ?", userID, "unread").
		Updates(map[string]any{
			"status":  "read",
			"read_at": &now,
		})

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
	userID := getCurrentUserID(c)
	if userID == 0 {
		res.FailAuth(c)
		return
	}
	var uri models.BindId
	if err := c.ShouldBindUri(&uri); err != nil {
		res.FailWithMsg(c, "参数错误")
		return
	}
	result := global.DB.Where("id = ? AND to_user_id = ?", uri.ID, userID).
		Delete(&models.Notification{})
	if result.RowsAffected == 0 {
		res.FailWithMsg(c, "通知不存在")
		return
	}

	// Push updated unread count
	if notifPusher != nil {
		var count int64
		global.DB.Model(&models.Notification{}).
			Where("to_user_id = ? AND status = ?", userID, "unread").
			Count(&count)
		notifPusher.Push(userID, map[string]any{
			"type":  "unread-count",
			"count": count,
		})
	}

	res.OkWithMsg(c, "已删除")
}
