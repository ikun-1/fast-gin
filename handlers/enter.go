package handlers

import (
	"fast-gin/handlers/captcha"
	"fast-gin/handlers/image"
	"fast-gin/handlers/rbac"
	"fast-gin/handlers/room"
	"fast-gin/handlers/signal"
	"fast-gin/handlers/user"
)

type Handler struct {
	User    user.User
	Captcha captcha.Captcha
	Image   image.Image
	RBAC    rbac.RBAC
	Room    room.Room
	Signal  signal.Signal
}

var Handlers = new(Handler)
