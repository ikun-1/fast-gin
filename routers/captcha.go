package routers

import (
	"fast-gin/handlers"

	"github.com/gin-gonic/gin"
)

func CaptchaRouter(g *gin.RouterGroup) {
	Captcha := handlers.Handlers.Captcha
	g.POST("captchas", Captcha.GenerateView)
}
