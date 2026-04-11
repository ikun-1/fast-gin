package views

import (
	"fast-gin/views/captcha"
	"fast-gin/views/image"
	"fast-gin/views/user"
)

type Handler struct {
	User    user.User
	Captcha captcha.Captcha
	Image   image.Image
}

var Handlers = new(Handler)
