package handlers

import (
	"fast-gin/handlers/captcha"
	"fast-gin/handlers/image"
	"fast-gin/handlers/user"
)

type Handler struct {
	User    user.User
	Captcha captcha.Captcha
	Image   image.Image
}

var Handlers = new(Handler)
