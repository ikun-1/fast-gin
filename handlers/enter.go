package handlers

import (
	"fast-gin/handlers/captcha"
	"fast-gin/handlers/image"
	"fast-gin/handlers/rbac"
	"fast-gin/handlers/user"
)

type Handler struct {
	User    user.User
	Captcha captcha.Captcha
	Image   image.Image
	RBAC    rbac.RBAC
}

var Handlers = new(Handler)
