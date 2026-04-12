package routers

import (
	"fast-gin/handlers"
	"fast-gin/middleware"
	"fast-gin/permissions"

	"github.com/gin-gonic/gin"
)

func ImageRouter(g *gin.RouterGroup) {
	Image := handlers.Handlers.Image
	g.POST("images/upload", middleware.AuthMiddleware, middleware.PermissionMiddleware(permissions.ImageUpload), Image.UploadView)
}
