package routers

import (
	"fast-gin/middleware"
	"fast-gin/views"

	"github.com/gin-gonic/gin"
)

func ImageRouter(g *gin.RouterGroup) {
	Image := views.Handlers.Image
	g.POST("images/upload", middleware.AuthMiddleware, Image.UploadView)
}
