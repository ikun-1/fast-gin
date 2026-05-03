package handlers

import (
	"fast-gin/handlers/captcha"
	"fast-gin/handlers/image"
	"fast-gin/handlers/meeting"
	"fast-gin/handlers/notification"
	"fast-gin/handlers/rbac"
	"fast-gin/handlers/recording"
	"fast-gin/handlers/stats"
	"fast-gin/handlers/user"
)

type Handler struct {
	User         user.User
	Captcha      captcha.Captcha
	Image        image.Image
	RBAC         rbac.RBAC
	Meeting      meeting.Meeting
	Recording    recording.Recording
	Stats        stats.Stats
	Notification notification.Notification
}

var Handlers = new(Handler)
