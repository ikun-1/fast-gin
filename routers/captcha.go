package routers

import (
	"fast-gin/views"

	"github.com/gin-gonic/gin"
)

func CaptchaRouter(g *gin.RouterGroup) {
	Captcha := views.Handlers.Captcha
	g.GET("captcha/generate", Captcha.GenerateView)
}
